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
    "landing": {
      "hero": {
        "releaseBadge": "v{version} 已发布",
        "releaseCta": "看看更新了什么",
        "tagline": "高智商模型，全程质检。",
        "client": {
          "label": "Agent 桌面客户端",
          "title": "所有 Agent，一个工作区，API 全接入。",
          "subtitle": "内置 Agent 免装 Node 和 CLI，Claude Code、Codex 等 Agent 都在一个窗口里跑，还能 AI 绘图。"
        },
        "api": {
          "label": "保质量 AI 中转站",
          "title": "一个 API Key 全接入。",
          "subtitle": "线路可用率持续探测，GPT 线路另用 Sol、Astra 指纹核对模型身份，异常即刻标红告警。"
        },
        "downloadFor": "下载 {platform} 版",
        "copyInstall": "复制 {platform} 安装命令",
        "switchPlatform": "改用 {platform} 版",
        "useApi": "直接使用 API",
        "startApi": "开始接入 API",
        "viewDocs": "查看接入文档",
        "installHint": "复制后在终端中粘贴运行",
        "terminal": {
          "title": "终端",
          "claudeComment": "# Claude Code",
          "openaiComment": "# Codex · OpenAI SDK · 各类兼容客户端",
          "caption": "只换 Base URL 和 Key，现有工具与代码零改动。"
        }
      },
      "features": {
        "client": {
          "titleLead": "Agent 干活需要一套系统，",
          "titleTail": "而不是一堆配置文件。",
          "subtitle": "登录一次，Claude Code、Codex、Grok 就都接好了；余额、线路和用量都在同一个窗口里。"
        },
        "api": {
          "titleLead": "模型有质检，",
          "titleTail": "接入只要一个 Key。",
          "subtitle": "每次探测、每次身份校验都有记录，出问题先标红再告警；Claude、GPT、Grok 共用一份余额和一套用量统计。"
        },
        "columns": {
          "quality": "保质量中转站",
          "client": "Agent 客户端"
        },
        "cards": {
          "quality": {
            "title": "模型身份校验",
            "body": "GPT 线路定时用 Sol 验证、Astra 指纹核对模型身份，连续对不上就判异常"
          },
          "probe": {
            "title": "线路实时探测",
            "body": "持续检测可用率和首字延迟，异常当场标红，运维即刻收到告警"
          },
          "builtinAgent": {
            "title": "内置 Agent",
            "body": "装好客户端就能用，不装 Node、不装 CLI，Claude、GPT、Grok、DeepSeek、Kimi 等模型随手切"
          },
          "allAgents": {
            "title": "所有 Agent 一个窗口",
            "body": "Claude Code、Codex、Grok、OpenCode、Pi 登录即自动接好，Amp、Cursor、Kimi Code 等也能在这里跑"
          },
          "images": {
            "title": "AI 绘图",
            "body": "一句话出图，或拖入图片直接改，出图前先报价，作品自动归档到图库"
          },
          "compatible": {
            "title": "原生协议兼容",
            "body": "OpenAI Chat / Responses 与 Anthropic Messages 原样可用"
          },
          "failover": {
            "title": "多账号自动切换",
            "body": "一条线路出问题自动换下一条，请求不中断"
          },
          "metered": {
            "title": "按量计费",
            "body": "用多少扣多少，没有最低消费，余额长期有效"
          },
          "invoice": {
            "title": "支持开发票",
            "body": "充值消费均可开具发票，团队报销省心"
          }
        }
      },
      "models": {
        "overline": "多模型 · 一处切换",
        "titleLead": "不止一个模型。",
        "titleTail": "每一步都用对的那个。",
        "clientSubtitle": "内置 Agent、Claude Code、Codex、Grok 在同一个客户端里随手切换，项目、文件和上下文都留在原地。",
        "apiSubtitle": "Claude、GPT、Grok 共用一个 Key 和一份余额，换模型只改一个参数。",
        "cta": "查看模型价格"
      },
      "closing": {
        "overline": "现在就开始",
        "clientTitleLead": "下载 {siteName}，",
        "clientTitleTail": "几分钟跑起第一个任务。",
        "apiTitleLead": "注册拿 Key，",
        "apiTitleTail": "几分钟发出第一个请求。",
        "note": "线路持续质检 · 按量计费 · 余额长期有效 · 支持开发票"
      },
      "footer": {
        "clientTagline": "保质量的 AI 中转站 × Agent 桌面客户端",
        "apiTagline": "保质量的 AI 模型网关",
        "product": "产品",
        "support": "支持",
        "download": "下载客户端",
        "pricing": "价格",
        "changelog": "更新日志",
        "docs": "接入文档",
        "console": "控制台"
      }
    },
    "hero": {
      "badge": "Claude Code / Codex 一键接入",
      "titleLeadPrimary": "具有价格竞争力的 AI 中转站",
      "titleLeadSecondary": "× Agent 桌面客户端",
      "description": "深度合一的 Claude Code · Codex · Grok · Pi 桌面 Agent 客户端，登录即路由，余额、分组倍率、在线率一目了然。",
      "primaryNote": "一个桌面端集中管理所有 Agent CLI",
      "downloadPrimary": "下载客户端",
      "installPrimary": "复制安装命令",
      "connectApi": "自行接入API",
      "startApi": "开始接入API",
      "badgeDiscount": "比官方 API",
      "tags": {
        "coding": "Claude Code",
        "agent": "Codex",
        "tools": "Grok · Pi · 更多",
        "more": "更多"
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
    "clientWorkflow": {
      "ariaLabel": "{siteName} 桌面客户端演示：内置 Agent 执行编码任务，在模型选择器里切换各家 Agent，再到画图页生成一张图",
      "sidebar": {
        "newTask": "新建任务",
        "search": "搜索",
        "images": "画图",
        "today": "今天",
        "taskTitle": "登录后跳回原页面",
        "working": "工作中 · {seconds} 秒",
        "justNow": "刚刚",
        "olderTask": "给首页加客户端下载按钮",
        "olderTime": "3 小时前",
        "email": "admin{'@'}cheaprouter.cc"
      },
      "transcript": {
        "prompt": "登录成功后跳回原来的页面，别总是回首页",
        "explored": "查阅 · 1 搜索，2 文件",
        "thought": "思考 · 持续了 3 秒",
        "edited": "已编辑",
        "ran": "已执行",
        "running": "正在执行",
        "working": "工作中 · {seconds} 秒",
        "workedFor": "已工作 {seconds} 秒",
        "reply": "改好了：登录前记下来源页面，登录成功后跳回去，没有来源时仍进控制台。测试全部通过。",
        "changedFiles": "更改了 {count} 个文件",
        "review": "审阅",
        "undo": "撤销"
      },
      "composer": {
        "placeholder": "做什么都可以…",
        "effort": "高",
        "access": "完全访问",
        "build": "构建"
      },
      "footer": {
        "project": "amadeus-web",
        "local": "本地"
      },
      "picker": {
        "search": "搜索模型…",
        "builtinAgent": "内置 Agent"
      },
      "image": {
        "title": "画图",
        "pictureCount": "共 {count} 张",
        "openFolder": "打开文件夹",
        "placeholder": "描述你想要的画面，或拖入、粘贴图片来改图…",
        "prompt": "霓虹灯下敲代码的橘猫，赛博朋克风",
        "addImages": "添加图片",
        "group": "分组：自动",
        "quality": "质量：高",
        "count": "1 张",
        "mode": "文生图",
        "estimate": "预计 $0.04",
        "generate": "生成",
        "drawing": "生成中 · {time}",
        "runningTask": "在网关上排队生成，关窗口也不丢",
        "gallery": {
          "sunset": "海边日落，胶片质感",
          "mountain": "雪山下的星空营地",
          "city": "雨夜霓虹街道，赛博朋克"
        }
      }
    },
    "download": {
      "commandCopied": "安装命令已复制，请在终端中运行"
    },
    "comparison": {
      "overline": "换个角度看",
      "title": "你可能正在用这些方式之一",
      "headers": {
        "feature": "你正在用的方式",
        "official": "常见做法的盲点",
        "us": "{siteName} 怎么做",
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
      "note": "不承诺\"全网最低价\"。{siteName} 关注的是总成本：一个充值入口、用多少扣多少、各线路用量分开看、额度和限速状态一目了然、故障原因清楚 — 让 AI 编码长期跑得起、看得清、出错时找得到原因。具体折扣随服务商和模型变动，最终扣费以控制台显示价格为准。",
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
    "footer": {
      "allRightsReserved": "保留所有权利。",
      "privacy": "隐私政策",
      "terms": "服务条款"
    },
    "privacy": {
      "title": "隐私政策",
      "backHome": "返回首页",
      "lastUpdated": "最后更新：2026 年 4 月",
      "intro": "本隐私政策说明 {siteName}（以下简称\"我们\"）在您使用本平台时如何收集、使用和保护您的信息。我们致力于保护您的隐私，尤其是代码与提示词内容。",
      "pledge": {
        "title": "代码与提示词不被存储，不用于训练",
        "body": "您通过本平台发送的所有代码内容、提示词及对话均直接转发至官方 API，{siteName} 不存储、不记录、不分析这些内容，也不会将其用于任何模型训练目的。",
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
          "p1": "您通过 API 发送的所有请求内容（包括代码、提示词、对话消息）均以透明代理方式直接转发至 Anthropic、OpenAI 等官方 API，{siteName} 不对请求内容进行存储或持久化记录。",
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
      "intro": "欢迎使用 {siteName}。在使用本平台前，请仔细阅读以下服务条款。注册或使用本服务即表示您同意受本条款约束。",
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
          "p1": "{siteName} 是一个 AI API 聚合代理平台，将您的请求转发至 Anthropic、OpenAI 等官方服务商。我们不对上游服务商的可用性、响应质量或内容负责。",
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
          "p1": "平台的界面、代码、品牌标识等知识产权归 {siteName} 所有。您通过 API 生成的内容归属依据上游服务商的政策确定，平台不主张对生成内容的权利。",
        },
        "disclaimer": {
          "title": "七、免责声明",
          "p1": "本服务按\"现状\"提供，不作任何明示或暗示的保证。在法律允许的最大范围内，{siteName} 不对任何间接、偶然、特殊或后果性损失承担责任，包括但不限于利润损失、数据丢失或业务中断。",
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
