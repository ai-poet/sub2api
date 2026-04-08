import { PrismaClient } from '@prisma/client';
import { PrismaPg } from '@prisma/adapter-pg';
import { getEnv } from '@/lib/config';

const globalForPrisma = globalThis as unknown as { prisma: PrismaClient };

export function getConnectionSchema(connectionString: string): string | undefined {
  try {
    const url = new URL(connectionString);
    const schema = url.searchParams.get('schema')?.trim();
    return schema || undefined;
  } catch {
    return undefined;
  }
}

export function getConfiguredDatabaseSchema(): string {
  return getConnectionSchema(getEnv().DATABASE_URL) || 'public';
}

function createPrismaClient() {
  const connectionString = getEnv().DATABASE_URL;
  const schema = getConnectionSchema(connectionString);
  const adapter = schema
    ? new PrismaPg({ connectionString }, { schema })
    : new PrismaPg({ connectionString });
  return new PrismaClient({ adapter });
}

export const prisma = globalForPrisma.prisma || createPrismaClient();

if (process.env.NODE_ENV !== 'production') globalForPrisma.prisma = prisma;
