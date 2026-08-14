// CheapRouter 落地页文案叠加层（home.* 含 privacy/terms），
// 由 cheap-router 分支原单体 zh.ts 抽取，deepMerge 在最后生效。
export default {
  "home": {
    "viewOnGithub": "在 GitHub 上查看",
    "viewDocs": "查看文档",
    "docs": "文档",
    "switchToLight": "切换到浅色模式",
    "switchToDark": "切换到深色模式",
    "dashboard": "控制台",
    "login": "登录",
    "getStarted": "立即开始",
    "goToDashboard": "进入控制台",
    "loginConsole": "登录控制台",
    "headerTagline": "Claude Code / Codex 一键接入",
    "heroSubtitle": "一键接入、按量充值、用量看得见",
    "heroDescription": "Claude Code 和 Codex 用同一个账户跑，开通快、价格友好、出错有解释。",
    "navModels": "模型",
    "navPricing": "价格",
    "navChangelog": "更新日志",
    "hero": {
      "badge": "Claude Code / Codex 一键接入",
      "titleLeadPrimary": "Claude Code 和 Codex",
      "titleLeadSecondary": "零门槛使用",
      "titleAccent": "",
      "titleTail": "",
      "description": "一键接入、按量充值、看清每一笔花费。Claude Code、Codex 与 GPT-5.5 共用一个账户、一份余额、一个用量面板，不用再为两套配置、两个接口地址、两份账单头疼。桌面端注册登录后自动帮你配置好 Claude Code 和 Codex，不用你再去复制粘贴 JSON 配置。",
      "primaryNote": "一个桌面端集中使用Claude Code和Codex",
      "downloadPrimary": "立即下载",
      "connectApi": "自行接入API",
      "badgeDiscount": "比官方 API",
      "tags": {
        "coding": "Claude Code",
        "agent": "Codex",
        "tools": "多账号 · 多模型"
      },
      "stats": {
        "setupValue": "1 步",
        "savings": "自动配置本地工具",
        "routesValue": "2 套",
        "models": "Claude / Codex 分开管理",
        "workspaceValue": "1 处",
        "minCost": "余额、用量、故障原因都在这里"
      },
      "perToken": "参考价格示意",
      "panel": {
        "title": "为日常编码准备的统一入口",
        "auto": "自动轮播",
        "requestLabel": "请求",
        "modelLabel": "模型",
        "routeLabel": "线路",
        "billLabel": "计费",
        "scenarios": {
          "completions": {
            "title": "一个入口搞定高级模型访问、额度控制和自动切换",
            "subtitle": "继续沿用常见 OpenAI 对话方式，把模型选择、计费和线路切换放到同一层。",
            "route": "多账号自动切换",
            "billing": "按量计费，花多少算多少"
          },
          "responses": {
            "title": "不只对话，也支持工具调用和长链路任务",
            "subtitle": "适合带工具、多步骤的复杂工作流，不用为了不同接口拆成多套服务。",
            "route": "工具调用 + 长链路任务",
            "billing": "按量计费，花多少算多少"
          },
          "messages": {
            "title": "Claude 对话也能走同一个入口",
            "subtitle": "保留 Claude 原生习惯，同时复用统一额度、线路切换和账单能力。",
            "route": "Claude 原生 + 统一入口",
            "billing": "按量计费，花多少算多少"
          }
        }
      }
    },
    "pricing": {
      "title": "按量付费",
      "subtitle": "对比官方 Claude Code / Codex API 服务",
      "highlight": "价格比官方 API 友好得多",
      "description": "用比官方更友好的价格跑高级模型，一个入口管 Claude Code 和 Codex，余额不限期，用多少扣多少。",
      "badge": "高级模型能力，不再被官方订阅价绑住",
      "barTitle": "按量计费，余额长期有效",
      "points": {
        "metered": "用多少扣多少，没有最低消费",
        "routing": "多账号自动切换，一条线路故障自动换另一条",
        "visibility": "近 5 小时 / 1 天 / 7 天的用量一目了然"
      }
    },
    "proofStrip": {
      "overline": "平台能力"
    },
    "clientShowcase": {
      "badge": "客户端预览",
      "title": "整合Claude Code和Codex双边能力的Agent客户端 国内下载即用",
      "description": "多个Agent通过Worktree并行工作 · Claude 和 Codex 同屏使用 · 文件变更追踪 · 消费透明",
      "pills": {
        "darkMode": "深色主题",
        "workspace": "Workspace 管理",
        "terminal": "内置终端",
        "parallel": "并行多开"
      },
      "cta": "注册获取通知",
      "ctaNote": "注册后可第一时间获取客户端更新",
      "downloadCta": "下载 {platform} 客户端",
      "downloadNote": "桌面客户端下载已开放，请选择适合当前设备的安装包"
    },
    "clientWorkflow": {
      "ariaLabel": "CheapRouter Agent 工作流动画预览",
      "windowTitle": "CheapRouter Client",
      "connected": "Daemon connected",
      "sidebarSubtitle": "本地 Agent 控制台",
      "workspaces": "Workspaces",
      "workspaceName": "新建工作区",
      "workspaceDocs": "client 样式源",
      "newWorktree": "新建 WorkTree",
      "creatingWorktree": "创建中",
      "agents": "Agents",
      "draftAgent": "Draft agent",
      "runningAgent": "Agent 正在执行",
      "branch": "main",
      "balance": "余额 ¥128.40",
      "terminal": "Terminal",
      "tabDraft": "New agent",
      "tabAgent": "Agent 工作流预览",
      "emptyTitle": "要在 {project} 中构建什么？",
      "emptyCopy": "桌面端保留你的本地代码，只把 Claude Code 和 Codex 的运行状态集中展示。",
      "composerTitle": "What should the agent build?",
      "provider": "Claude",
      "group": "Claude",
      "model": "Opus 4.8 1M",
      "mode": "Bypass",
      "thinking": "Medium",
      "prompt": "创建工作区，接入计费看板，并验证 Claude Agent 执行流。",
      "runningStatus": "Agent 正在用 Claude / Opus 4.8 1M 流式输出...",
      "workingRead": "正在读取工作区文件",
      "workingEdit": "正在修改看板与状态",
      "workingVerify": "正在运行验证",
      "streamThinking": "正在读取工作区文件，并规划计费看板改造...",
      "requestTitle": "Agent request",
      "requestBody": "允许 Agent 检查本地文件并运行验证命令。",
      "permissionQuestion": "允许继续执行吗？",
      "requestDeny": "Deny",
      "requestApproved": "Allow",
      "toolInspect": "读取工作区文件",
      "toolEdit": "应用看板改造",
      "toolTerminal": "运行验证命令",
      "streamStepOne": "已更新看板卡片，并把用量摘要接入工作区状态。",
      "streamStepTwo": "已在桌面工作流里验证 Agent 请求、文件变更和终端状态。",
      "streamComplete": "完成：Claude Agent 工作流已执行",
      "replay": "重新播放",
      "filesChanged": "Files changed",
      "terminalRun": "Checks",
      "typecheckDone": "typecheck passed",
      "buildDone": "production build ready",
      "spendTitle": "透明消费",
      "spendInput": "Input tokens",
      "spendOutput": "Output tokens"
    },
    "download": {
      "badge": "客户端下载",
      "title": "下载 CheapRouter 桌面客户端",
      "description": "安装桌面端后自动帮你配置 Claude Code 和 Codex，安全复用已有本地设置，并把用量和状态集中在一个入口。",
      "privacyCode": "自动写入本地配置文件，已有设置不会被覆盖",
      "privacyKey": "桌面、Web、移动端共享一份用量和故障信息",
      "cta": "下载",
      "recommended": "推荐用于当前设备",
      "platforms": {
        "mac": {
          "sub": "Apple Silicon 与 Intel"
        },
        "windows": {
          "sub": "x64 / ARM64"
        },
        "linux": {
          "sub": ".deb / .rpm / AppImage"
        }
      }
    },
    "value": {
      "overline": "为什么选 CheapRouter",
      "title": "让 Claude Code 和 Codex 跑起来这件事，更省心",
      "description": "不是再做一个 IDE，也不是又一个 API 中转站。CheapRouter 把开通账户、充值、配置工具、看用量、查故障这些散落的事，合成一条顺手的路径。",
      "items": {
        "economics": {
          "eyebrow": "01 / 接入快",
          "title": "一键写好本地配置",
          "description": "桌面端注册登录后自动帮你配置好 Claude Code 和 Codex，不用你再去复制粘贴 JSON 配置，不会覆盖你已有的设置。"
        },
        "reliability": {
          "eyebrow": "02 / 花费看得见",
          "title": "一个余额面板看清两路消费",
          "description": "Claude 和 Codex 各自用了多少、还剩多少额度、当前限速状态如何，都在同一处展示，不用再在多个平台之间切。"
        },
        "control": {
          "eyebrow": "03 / all in one",
          "title": "使用我们的服务有 SLA 保障",
          "description": "全线服务可用性、响应速度、定价策略统一管理。出问题时不抓瞎，自动切换到可用线路，或给出明确的恢复建议。"
        }
      }
    },
    "comparison": {
      "overline": "换个角度看",
      "title": "你可能正在用这些方式之一",
      "description": "",
      "headers": {
        "feature": "你正在用的方式",
        "official": "常见做法的盲点",
        "us": "CheapRouter 怎么做"
      },
      "items": {
        "pricing": {
          "feature": "官方账号 + 手工配置",
          "official": "每个工具一个账号，多套配置、多份账单，需要自己手动维护本地设置",
          "us": "一个账户开通后，自动配置好两套工具，余额跨工具共用"
        },
        "models": {
          "feature": "本地切换脚本",
          "official": "能切换接口地址，但不知道余额还剩多少、当前额度或限速是什么状态",
          "us": "切换之后，余额、额度、近 5 小时 / 1 天 / 7 天用量和线路健康度直接展示"
        },
        "stability": {
          "feature": "工作流",
          "official": "Cursor、Windsurf 等云端 IDE 需要你把代码上传到它们的服务器",
          "us": "比 Cursor、Windsurf 更尊重你本地的 Claude Code 和 Codex 工作流，MCP、Skill 和 Workflow 无需额外配置"
        }
      }
    },
    "pricingTable": {
      "overline": "价格优势",
      "title": "主流编码模型，价格比官方友好得多",
      "description": "Claude Code、Codex、GPT-5.5 等主流编码模型，比官方 API 便宜很多。一个账户、一份余额，用多少扣多少，余额长期有效。",
      "badge": "目标",
      "badgeValue": "总成本更低",
      "badgeSavings": "低至",
      "badgeSavingsValue": "{discount} 折",
      "currencyNote": "表中价格按 1 USD ≈ ¥{rate} 换算展示，实际结算以美元计价。",
      "table": {
        "model": "模型",
        "group": "分组",
        "input": "输入 / 1M",
        "output": "输出 / 1M",
        "cacheWrite": "缓存写 / 1M",
        "cacheRead": "缓存读 / 1M",
        "discountHeader": "折扣",
        "discount": "{discount} 折",
        "perRequest": "{price} / 次",
        "perImage": "{price} / 张",
        "modelCount": "{count} 个模型",
        "expand": "展开全部（还有 {count} 个）",
        "collapse": "收起"
      },
      "cards": {
        "claude": {
          "tag": "Claude 系列",
          "title": "Claude Code 主力",
          "description": "Claude Sonnet 4.6 / Opus 4.7 / Haiku 4.5 全系覆盖，日常编码、重构、审查交给 Claude Code，每条线路的消费单独计算。"
        },
        "codex": {
          "tag": "Codex / GPT 系列",
          "title": "Codex CLI 与 GPT-5.5 都在这",
          "description": "GPT-5.5 与 GPT-5.4 / GPT-5.3 Codex 同步可用，本地 Codex 自动配置好，不用手动改设置。"
        },
        "compatible": {
          "tag": "OpenAI 兼容 · 多家",
          "title": "其它兼容模型也在同一个网关",
          "description": "Gemini、GLM、Qwen 等兼容模型同一个入口接入，价格以控制台当前显示为准，故障原因同样清楚明了。"
        }
      },
      "note": "不承诺\"全网最低价\"。CheapRouter 关注的是总成本：一个充值入口、用多少扣多少、各线路用量分开看、额度和限速状态一目了然、故障原因清楚 — 让 AI 编码长期跑得起、看得清、出错时找得到原因。具体折扣随服务商和模型变动，最终扣费以控制台显示价格为准。"
    },
    "providers": {
      "title": "把高级模型与本机编码工作流放进同一层服务",
      "description": "围绕高频 coding agents 而不是通用模型陈列，保留 Claude Code、Codex 和 OpenAI 兼容调用方式。",
      "supported": "已支持",
      "claude": "Claude",
      "claudeCode": "Claude Code",
      "gpt": "GPT",
      "codex": "Codex",
      "gemini": "Gemini",
      "openaiCompatible": "OpenAI 兼容"
    },
    "trust": {
      "overline": "用得放心",
      "title": "钱花在哪、为什么不能用，都看得见",
      "description": "中转服务最常见的问题是不透明：余额不知道还剩多少、报错不知道哪里坏。CheapRouter 把这些信息显式做出来，让你不用猜。",
      "cards": {
        "gateway": {
          "title": "每条线路分开算",
          "description": "Claude Code 和 Codex 各用各的配置，不会互相覆盖；每条线路的消费、近 5 小时 / 1 天 / 7 天用量、剩余额度都独立展示。"
        },
        "resilience": {
          "title": "服务商出问题，提前知道",
          "description": "每个服务商的可用状态、响应速度、价格都摆在台面上，出问题时不抓瞎，知道该切到另一条线路还是等自动恢复。"
        },
        "visibility": {
          "title": "all in one",
          "description": "使用我们的服务有 SLA 保障。"
        }
      },
      "trackers": {
        "routing": "按量计费",
        "billing": "余额长期有效",
        "visibility": "多账号切换"
      }
    },
    "cta": {
      "eyebrow": "开始使用",
      "title": "Claude Code 和 Codex，一次配齐",
      "description": "注册账户、充值、自动配置好本地工具 — 三步之后就能开始跑。Claude 全系、Codex 与 GPT-5.5 都在同一个余额下，控制台里随时看清楚每一笔花费走的是哪条线路。",
      "button": "开始配置",
      "stat": "一键接入 · 按量计费 · 用量透明 · 价格友好"
    },
    "footer": {
      "allRightsReserved": "保留所有权利。",
      "privacy": "隐私政策",
      "terms": "服务条款"
    },
    "privacy": {
      "title": "隐私政策",
      "backHome": "返回首页",
      "lastUpdated": "最后更新：2026 年 4 月",
      "intro": "本隐私政策说明 CheapRouter（以下简称\"我们\"）在您使用本平台时如何收集、使用和保护您的信息。我们致力于保护您的隐私，尤其是代码与提示词内容。",
      "pledge": {
        "title": "代码与提示词不被存储，不用于训练",
        "body": "您通过本平台发送的所有代码内容、提示词及对话均直接转发至官方 API，CheapRouter 不存储、不记录、不分析这些内容，也不会将其用于任何模型训练目的。"
      },
      "sections": {
        "collection": {
          "title": "一、我们收集哪些信息",
          "p1": "为提供服务，我们仅收集必要的信息：",
          "i1": "账户信息：注册时提供的邮箱地址",
          "i2": "使用记录：token 消耗量、请求次数、模型类型（不含请求内容）",
          "i3": "支付信息：充值金额与交易记录（不含银行卡号等敏感金融数据）",
          "i4": "日志信息：IP 地址、请求时间戳，用于安全防护与故障排查"
        },
        "use": {
          "title": "二、信息的使用方式",
          "p1": "收集的信息仅用于以下目的：",
          "i1": "提供和维护 API 代理服务",
          "i2": "计算 token 消耗并生成账单明细",
          "i3": "发送服务通知（如余额不足提醒）",
          "i4": "保障账户安全，防止滥用"
        },
        "code": {
          "title": "三、代码与提示词保护",
          "p1": "您通过 API 发送的所有请求内容（包括代码、提示词、对话消息）均以透明代理方式直接转发至 Anthropic、OpenAI 等官方 API，CheapRouter 不对请求内容进行存储或持久化记录。",
          "p2": "我们不会将您的代码或提示词用于训练、微调或评估任何机器学习模型，也不会与第三方共享这些内容。"
        },
        "apikey": {
          "title": "四、API Key 安全",
          "p1": "您在本平台生成的 API Key 在服务器端加密存储。用于调用上游模型的官方 API 凭证由平台统一管理，不会暴露给用户。建议您妥善保管本平台 API Key，如发现异常请立即在控制台重置。"
        },
        "third": {
          "title": "五、第三方服务",
          "p1": "本平台将 API 请求转发至以下第三方服务商，这些服务商有其独立的隐私政策：",
          "i1": "Anthropic（Claude 系列模型）",
          "i2": "OpenAI（GPT / Codex 系列模型）"
        },
        "retention": {
          "title": "六、数据保留",
          "p1": "账户信息在您主动注销前保留。使用记录（token 数量、时间戳，不含内容）保留 90 天用于账单核对。您可随时在账户设置中申请删除账户及相关数据。"
        },
        "rights": {
          "title": "七、您的权利",
          "p1": "您对个人数据享有以下权利：",
          "i1": "查阅权：查看我们持有的关于您的数据",
          "i2": "更正权：更新不准确的账户信息",
          "i3": "删除权：申请注销账户并删除相关数据"
        },
        "changes": {
          "title": "八、政策变更",
          "p1": "如本政策发生重大变更，我们将通过站内通知或邮件提前告知。继续使用本服务即表示您接受更新后的政策。"
        }
      },
      "contact": "如对本隐私政策有任何疑问，请通过平台内的客服渠道联系我们。"
    },
    "terms": {
      "title": "服务条款",
      "lastUpdated": "最后更新：2026 年 4 月",
      "intro": "欢迎使用 CheapRouter。在使用本平台前，请仔细阅读以下服务条款。注册或使用本服务即表示您同意受本条款约束。",
      "sections": {
        "eligibility": {
          "title": "一、适用资格",
          "p1": "本服务面向具有完全民事行为能力的个人及合法注册的商业实体。未满 18 周岁者须在监护人同意下使用。使用本服务即表示您声明并保证您有权力接受本条款。"
        },
        "account": {
          "title": "二、账户责任",
          "p1": "您对账户安全及账户下的所有活动负全部责任：",
          "i1": "请妥善保管登录凭证和 API Key，勿与他人共享",
          "i2": "如发现账户异常或未授权访问，请立即联系我们",
          "i3": "禁止转让、出售或共享账户"
        },
        "service": {
          "title": "三、服务说明",
          "p1": "CheapRouter 是一个 AI API 聚合代理平台，将您的请求转发至 Anthropic、OpenAI 等官方服务商。我们不对上游服务商的可用性、响应质量或内容负责。",
          "p2": "我们保留在提前通知的情况下修改、暂停或终止服务的权利。因上游服务商故障、维护或不可抗力造成的服务中断不在我们的责任范围内。"
        },
        "billing": {
          "title": "四、计费与退款",
          "p1": "本平台采用预付费模式：",
          "i1": "充值后的余额不支持退款，请按需充值",
          "i2": "Token 消耗以平台系统记录为准，实时扣除",
          "i3": "如因平台故障造成异常扣费，可联系客服申请核查补偿"
        },
        "prohibited": {
          "title": "五、禁止行为",
          "p1": "使用本服务时，您不得：",
          "i1": "将本平台用于生成违法、有害、歧视性或侵权内容",
          "i2": "尝试破解、绕过或滥用平台的速率限制和安全机制",
          "i3": "将账户或 API Key 转售、分发或用于商业代理服务（未经授权）",
          "i4": "违反上游模型服务商（Anthropic、OpenAI 等）的使用政策"
        },
        "ip": {
          "title": "六、知识产权",
          "p1": "平台的界面、代码、品牌标识等知识产权归 CheapRouter 所有。您通过 API 生成的内容归属依据上游服务商的政策确定，平台不主张对生成内容的权利。"
        },
        "disclaimer": {
          "title": "七、免责声明",
          "p1": "本服务按\"现状\"提供，不作任何明示或暗示的保证。在法律允许的最大范围内，CheapRouter 不对任何间接、偶然、特殊或后果性损失承担责任，包括但不限于利润损失、数据丢失或业务中断。"
        },
        "termination": {
          "title": "八、账户终止",
          "p1": "我们保留在您违反本条款时暂停或终止账户的权利，恕不另行通知。您也可随时在账户设置中注销账户。账户注销后，剩余余额将不予退还。"
        },
        "changes": {
          "title": "九、条款变更",
          "p1": "我们可能不时更新本条款。重大变更将提前通过站内通知或邮件告知。继续使用本服务即视为接受更新后的条款。"
        }
      },
      "contact": "如对本服务条款有任何疑问，请通过平台内的客服渠道联系我们。"
    }
  }
}
