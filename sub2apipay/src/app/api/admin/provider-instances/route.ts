import { NextRequest, NextResponse } from 'next/server';
import { verifyAdminToken, unauthorizedResponse } from '@/lib/admin-auth';
import { prisma } from '@/lib/db';
import { encrypt, decrypt } from '@/lib/crypto';

/** Fields whose values should be masked when returning to the client */
const SENSITIVE_PATTERNS = ['key', 'pkey', 'secret', 'private', 'password'];

function isSensitiveField(fieldName: string): boolean {
  const lower = fieldName.toLowerCase();
  return SENSITIVE_PATTERNS.some((p) => lower.includes(p));
}

/**
 * Decrypt config JSON and mask sensitive fields (show only last 4 chars).
 */
function decryptAndMaskConfig(encryptedConfig: string): Record<string, string> {
  const config: Record<string, string> = JSON.parse(decrypt(encryptedConfig));
  const masked: Record<string, string> = {};
  for (const [key, value] of Object.entries(config)) {
    if (isSensitiveField(key) && value && value.length > 4) {
      masked[key] = '*'.repeat(value.length - 4) + value.slice(-4);
    } else if (isSensitiveField(key) && value) {
      masked[key] = '****';
    } else {
      masked[key] = value;
    }
  }
  return masked;
}

/**
 * 解密失败的实例不应该让整个列表 500。
 *
 * config 用 SHA256(JWT_SECRET) 加密，轮换过 JWT_SECRET 之后旧密文就永久解不开了。
 * 之前一行坏掉会抛穿整个 map，管理员连页面都打不开，也就无从删除或重建那一行——
 * 恰好在最需要后台的时候把后台锁死。这里改为单行降级，返回标记让前端提示重配。
 * ensureDBProviders 早就是这么做的（逐个 try/catch 后 skip）。
 */
function safeDecryptAndMaskConfig(
  encryptedConfig: string,
): { config: Record<string, string>; decryptFailed: boolean } {
  try {
    return { config: decryptAndMaskConfig(encryptedConfig), decryptFailed: false };
  } catch {
    return { config: {}, decryptFailed: true };
  }
}

// GET: List all instances (optionally filter by providerKey)
export async function GET(request: NextRequest) {
  if (!(await verifyAdminToken(request))) return unauthorizedResponse(request);

  try {
    const providerKey = request.nextUrl.searchParams.get('providerKey');

    const instances = await prisma.paymentProviderInstance.findMany({
      where: providerKey ? { providerKey } : undefined,
      orderBy: { sortOrder: 'asc' },
    });

    const result = instances.map((inst) => {
      const { config, decryptFailed } = safeDecryptAndMaskConfig(inst.config);
      let limits = null;
      try {
        limits = inst.limits ? JSON.parse(inst.limits) : null;
      } catch {
        // 同理：一行 limits 存坏了也不该拖垮整页。
      }
      return { ...inst, config, limits, config_decrypt_failed: decryptFailed };
    });

    const brokenCount = result.filter((inst) => inst.config_decrypt_failed).length;
    if (brokenCount > 0) {
      console.warn(
        `[payment] ${brokenCount} provider instance(s) could not be decrypted; ` +
          'this usually means JWT_SECRET was rotated. They must be deleted and recreated.',
      );
    }

    return NextResponse.json({ instances: result });
  } catch (error) {
    console.error('Failed to list provider instances:', error instanceof Error ? error.message : String(error));
    return NextResponse.json({ error: '获取支付实例列表失败' }, { status: 500 });
  }
}

// POST: Create a new instance
export async function POST(request: NextRequest) {
  if (!(await verifyAdminToken(request))) return unauthorizedResponse(request);

  try {
    const body = await request.json();
    const { providerKey, name, config, enabled, sortOrder, supportedTypes, limits, refundEnabled } = body;

    // Validate required fields
    if (!providerKey || typeof providerKey !== 'string') {
      return NextResponse.json({ error: '缺少必填字段: providerKey' }, { status: 400 });
    }
    if (!name || typeof name !== 'string' || name.trim() === '') {
      return NextResponse.json({ error: '缺少必填字段: name' }, { status: 400 });
    }
    if (!config || typeof config !== 'object') {
      return NextResponse.json({ error: '缺少必填字段: config (必须是对象)' }, { status: 400 });
    }

    const validProviders = ['easypay', 'alipay', 'wxpay', 'stripe'];
    if (!validProviders.includes(providerKey)) {
      return NextResponse.json({ error: `无效的 providerKey，可选值: ${validProviders.join(', ')}` }, { status: 400 });
    }

    if (sortOrder !== undefined && (!Number.isInteger(sortOrder) || sortOrder < 0)) {
      return NextResponse.json({ error: 'sortOrder 必须是非负整数' }, { status: 400 });
    }

    // Encrypt config before storing
    const encryptedConfig = encrypt(JSON.stringify(config));

    const instance = await prisma.paymentProviderInstance.create({
      data: {
        providerKey,
        name: name.trim(),
        config: encryptedConfig,
        supportedTypes: supportedTypes ?? '',
        enabled: enabled ?? true,
        sortOrder: sortOrder ?? 0,
        limits: limits ? JSON.stringify(limits) : null,
        refundEnabled: refundEnabled === true,
      },
    });

    return NextResponse.json(
      {
        ...instance,
        config: decryptAndMaskConfig(instance.config),
      },
      { status: 201 },
    );
  } catch (error) {
    console.error('Failed to create provider instance:', error instanceof Error ? error.message : String(error));
    return NextResponse.json({ error: '创建支付实例失败' }, { status: 500 });
  }
}
