// fork 自有的翻译键（上游 locale 中不存在）。
// 单独成文件：同步上游 locale 模块时不会与之冲突，index.ts 里深合并生效。
export default {
  "common": {
    "retry": "重试"
  },
  "home": {
    "loginConsole": "登录控制台",
    "headerTagline": "Claude Code / Codex 一键接入",
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
      "startApi": "开始接入API",
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
      "title": "下载桌面客户端",
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
      "overline": "为什么选择我们",
      "title": "让 Claude Code 和 Codex 跑起来这件事，更省心",
      "description": "不是再做一个 IDE，也不是又一个 API 中转站。把开通账户、充值、配置工具、看用量、查故障这些散落的事，合成一条顺手的路径。",
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
      "description": ""
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
      "note": "不承诺\"全网最低价\"。我们关注的是总成本：一个充值入口、用多少扣多少、各线路用量分开看、额度和限速状态一目了然、故障原因清楚 — 让 AI 编码长期跑得起、看得清、出错时找得到原因。具体折扣随服务商和模型变动，最终扣费以控制台显示价格为准。"
    },
    "providers": {
      "claudeCode": "Claude Code",
      "gpt": "GPT",
      "codex": "Codex",
      "openaiCompatible": "OpenAI 兼容"
    },
    "trust": {
      "overline": "用得放心",
      "title": "钱花在哪、为什么不能用，都看得见",
      "description": "中转服务最常见的问题是不透明：余额不知道还剩多少、报错不知道哪里坏。我们把这些信息显式做出来，让你不用猜。",
      "cards": {
        "gateway": {
          "title": "每条线路分开算",
          "description": "每条线路单独计费，避免一条故障拖垮整体账单。"
        },
        "resilience": {
          "title": "稳定切换",
          "description": "当单个上游限制、波动或短时异常出现时，平台负责承担切换与缓冲。"
        },
        "visibility": {
          "title": "故障原因看得见",
          "description": "出错时给出清晰原因和恢复建议，而不是黑盒报错。"
        }
      },
      "trackers": {
        "routing": "粘性会话",
        "billing": "按量计费",
        "visibility": "用量明细"
      }
    },
    "cta": {
      "eyebrow": "READY TO START",
      "stat": "统一入口 · 按量付费 · 面向日常开发"
    }
  },
  "changelog": {
    "title": "更新日志",
    "subtitle": "追踪最新变化与改进。",
    "emptyTitle": "暂无更新",
    "emptyDesc": "稍后回来看看最新的变化与改进。"
  },
  "modelStatus": {
    "title": "模型运行状态",
    "description": "查看你当前可访问分组的实时运行状态与近期可用率",
    "featureDisabledTitle": "模型运行状态未开放",
    "featureDisabledDescription": "管理员尚未开启此功能。",
    "totalGroups": "监测分组",
    "healthyGroups": "运行正常",
    "degradedGroups": "响应变慢",
    "downGroups": "异常分组",
    "lastUpdated": "最近刷新",
    "autoRefresh": "列表每 30 秒自动刷新一次",
    "latestProbe": "最近检测",
    "latestLatency": "最新延迟",
    "availability24": "24h 可用率",
    "availability7": "7d 可用率",
    "uptimeTimeline": "最近 24 次检测",
    "recentChecksHint": "每个色块对应一次真实检测结果",
    "latestResult": "最近结果",
    "openDetails": "查看详情",
    "emptyTitle": "暂无可展示的分组",
    "emptyDescription": "当前没有你可访问且已启用监测的分组。",
    "waiting": "等待探测",
    "waitingForProbe": "尚未进行首次探测",
    "notAvailable": "暂无",
    "noDescription": "未填写分组说明",
    "loadFailed": "加载模型运行状态失败",
    "detailLoadFailed": "加载状态详情失败",
    "historyTitle": "历史趋势",
    "historyDescription": "按时间桶聚合的可用率与最近状态。",
    "eventsTitle": "最近事件",
    "eventsDescription": "仅记录稳定状态变化事件。",
    "noHistory": "当前时间范围内暂无历史数据",
    "noEvents": "暂无状态事件",
    "period24h": "24 小时",
    "period7d": "7 天",
    "bucketAvailability": "桶可用率",
    "sampleCount": "采样次数",
    "avgLatency": "平均延迟",
    "httpCode": "HTTP 状态",
    "subStatus": "子状态",
    "statuses": {
      "up": "正常",
      "degraded": "降级",
      "down": "异常",
      "unknown": "未知"
    },
    "eventTypes": {
      "up": "恢复",
      "down": "中断"
    }
  },
  "nav": {
    "integrationGuide": "使用教程",
    "referralSettings": "推荐设置",
    "dataManagement": "数据管理",
    "paymentManagement": "支付管理",
    "referral": "推荐邀请",
    "modelMirror": "模型照妖镜",
    "modelCatalog": "模型广场",
    "modelStatus": "模型运行状态",
    "sora": "Sora 创作",
    "communityGroup": "加入交流群",
    "communityGroupTooltip": "扫码加入交流群",
    "communityGroupScanHint": "使用手机扫码加入",
    "communityGroupJoin": "点击加入"
  },
  "auth": {
    "referralCodeLabel": "推荐码",
    "referralCodePlaceholder": "输入推荐码（可选）",
    "github": {
      "signIn": "使用 GitHub 登录",
      "orContinue": "或使用邮箱密码继续",
      "callbackTitle": "正在完成登录",
      "callbackProcessing": "正在验证登录信息，请稍候...",
      "callbackHint": "如果页面未自动跳转，请返回登录页重试。",
      "callbackMissingToken": "登录信息缺失，请返回重试。",
      "backToLogin": "返回登录",
      "invitationRequired": "该 GitHub 账号尚未注册，站点已开启邀请码注册，请输入邀请码以完成注册。",
      "invalidPendingToken": "注册凭证已失效，请重新使用 GitHub 登录。",
      "completeRegistration": "完成注册",
      "completing": "正在完成注册...",
      "completeRegistrationFailed": "注册失败，请检查邀请码后重试。"
    }
  },
  "integrationGuide": {
    "title": "使用教程",
    "description": "按平台查看客户端接入方式，并直接复制可用配置",
    "caption": "多平台接入教程",
    "intro": "这里会按平台整理常见客户端的接入方式。每个平台都可以单独选择一个已有 API Key，页面会直接生成可复制的配置内容。",
    "bindingNote": "一个 API Key 绑定一个分组平台，不能直接通用于所有平台。没有可用密钥的平台会显示示例配置，等你创建对应平台的密钥后即可直接使用。",
    "liveBadge": "真实配置",
    "exampleBadge": "示例模式",
    "keyLabel": "当前 API Key",
    "selectKey": "选择此平台的 API Key",
    "keyHelp": "已自动填入真实密钥，可直接复制使用。",
    "noKeysForPlatform": "当前平台还没有可用 API Key。",
    "exampleOption": "显示示例配置",
    "exampleDescription": "当前展示的是占位示例。创建并分配对应平台的密钥后，这里会自动切换为真实配置。",
    "failedToLoadKeys": "加载使用教程所需的 API 密钥失败",
    "failedToLoadSettings": "加载使用教程所需的公共设置失败",
    "platforms": {
      "anthropic": "Anthropic",
      "openai": "OpenAI"
    }
  },
  "redeem": {
    "referralReward": "推荐奖励"
  },
  "referral": {
    "title": "推荐邀请",
    "description": "邀请好友注册并充值，双方均可获得奖励",
    "yourLink": "您的推荐链接",
    "copyLink": "复制链接",
    "copied": "已复制",
    "code": "推荐码",
    "totalInvited": "总邀请数",
    "rewarded": "已奖励",
    "pending": "待奖励",
    "totalEarned": "累计获得",
    "history": "推荐历史",
    "referee": "被推荐人",
    "status": "状态",
    "reward": "奖励",
    "time": "时间",
    "noHistory": "暂无推荐记录",
    "statusRewarded": "已奖励",
    "statusPending": "待充值",
    "days": "天",
    "loadFailed": "加载推荐信息失败",
    "copyFailed": "复制失败，请手动复制",
    "rulesTitle": "推荐奖励规则",
    "rule1": "分享您的推荐链接给好友，好友通过链接注册即建立推荐关系",
    "rule2": "被推荐人首次充值后，双方均可获得奖励",
    "rule3": "奖励将自动发放到账户余额或订阅时长",
    "ruleReferrerReward": "推荐人奖励",
    "ruleRefereeReward": "被推荐人奖励"
  },
  "admin": {
    "users": {
      "typeReferralReward": "余额（推荐奖励）"
    },
    "groups": {
      "columns": {
        "runtimeStatus": "运行状态"
      },
      "runtimeStatus": {
        "action": "运行状态配置",
        "title": "{name} 运行状态配置",
        "titlePlain": "运行状态配置",
        "enabled": "启用分组运行状态监测",
        "enabledHint": "关闭后该分组不会参与后台探测，也不会展示给普通用户。",
        "probeModel": "探测模型",
        "probeModelPlaceholder": "例如：gpt-4.1-mini",
        "probePrompt": "探测提示词",
        "probePromptPlaceholder": "输入一段用于最小化探测的提示词",
        "validationMode": "校验方式",
        "validationModes": {
          "nonEmpty": "响应非空",
          "keywordsAny": "命中任意关键词",
          "keywordsAll": "命中全部关键词"
        },
        "expectedKeywords": "关键词列表",
        "expectedKeywordsPlaceholder": "多个关键词可用逗号或换行分隔",
        "expectedKeywordsHint": "仅在关键词校验模式下生效，后端会自动去重。",
        "intervalSeconds": "探测频率（秒）",
        "timeoutSeconds": "超时阈值（秒）",
        "slowLatencyMs": "慢请求阈值（毫秒）",
        "latestResult": "最近一次结果预览",
        "latestResultHint": "可查看最近一次稳定状态和原始探测结果摘要。",
        "latestResultEmpty": "尚未产生探测结果，保存配置或点击“立即探测”后会显示。",
        "currentStatus": "当前状态",
        "checkedAt": "检测时间",
        "latency": "延迟",
        "httpCode": "HTTP",
        "subStatus": "子状态",
        "responseExcerpt": "响应摘录",
        "errorDetail": "错误详情",
        "save": "保存配置",
        "saved": "运行状态配置已保存",
        "failedToLoad": "加载运行状态配置失败",
        "failedToSave": "保存运行状态配置失败",
        "probeNow": "立即探测",
        "probing": "探测中...",
        "probeSucceeded": "立即探测已完成",
        "probeFailed": "立即探测失败",
        "disabled": "未启用",
        "waiting": "等待探测",
        "notConfigured": "未配置",
        "footerHint": "“立即探测”会先保存当前配置，再立即执行一次探测。"
      },
      "openaiMessages": {
        "defaultModel": "默认映射模型",
        "defaultModelPlaceholder": "例如: gpt-4.1",
        "defaultModelHint": "当账号未配置模型映射时，所有请求模型将映射到此模型"
      }
    },
    "accounts": {
      "deleteConfirmMessage": "确定要删除账号 '{name}' 吗？",
      "refreshCookie": "刷新 Cookie",
      "testAccount": "测试账号",
      "types": {
        "api_key": "API Key",
        "cookie": "Cookie"
      },
      "form": {
        "nameLabel": "账号名称",
        "namePlaceholder": "请输入账号名称",
        "platformLabel": "平台",
        "selectPlatform": "选择平台",
        "typeLabel": "类型",
        "selectType": "选择类型",
        "credentialsLabel": "凭证",
        "credentialsPlaceholder": "请输入 Cookie 或 API Key",
        "priorityLabel": "优先级",
        "priorityHint": "数值越小优先级越高",
        "weightLabel": "权重",
        "weightHint": "用于负载均衡的权重值",
        "statusLabel": "状态"
      },
      "filters": {
        "platform": "平台",
        "allPlatforms": "全部平台",
        "type": "类型",
        "allTypes": "全部类型",
        "status": "状态",
        "allStatuses": "全部状态"
      },
      "saving": "保存中...",
      "refreshing": "刷新中...",
      "noAccounts": "暂无账号",
      "noAccountsDescription": "添加 AI 平台账号以开始使用 API 网关。",
      "accountCreatedSuccess": "账号添加成功",
      "accountUpdatedSuccess": "账号更新成功",
      "accountDeletedSuccess": "账号删除成功",
      "cookieRefreshedSuccess": "Cookie 刷新成功",
      "testSuccess": "账号测试通过",
      "failedToSave": "保存账号失败",
      "sendingGeminiImageRequest": "发送 Gemini 生图测试请求...",
      "geminiImagePromptLabel": "生图提示词",
      "geminiImagePromptPlaceholder": "例如：生成一只戴宇航员头盔的橘猫，像素插画风格，纯色背景。",
      "geminiImagePromptDefault": "Generate a cute orange cat astronaut sticker on a clean pastel background.",
      "geminiImageTestHint": "选择 Gemini 图片模型后，这里会直接发起生图测试，并在下方展示返回图片。",
      "geminiImageTestMode": "模式：Gemini 生图测试",
      "geminiImagePreview": "生成结果：",
      "geminiImageReceived": "已收到第 {count} 张测试图片"
    },
    "redeem": {
      "types": {
        "referral_reward": "推荐奖励"
      }
    },
    "ops": {
      "errorDetail": {
        "pinnedToOriginalAccountId": "固定到原 account_id",
        "missingUpstreamRequestBody": "缺少上游请求体",
        "failedToLoadRetryHistory": "加载重试历史失败",
        "unsupportedRetryMode": "不支持的重试模式",
        "classificationKeys": {
          "retryable": "可重试",
          "resolvedRetryId": "解决重试ID",
          "retryCount": "重试次数"
        },
        "retryMeta": {
          "used": "使用账号",
          "success": "成功",
          "pinned": "固定账号"
        },
        "notRetryable": "此错误不建议重试",
        "retry": "重试",
        "retryClient": "重试（客户端）",
        "retryUpstream": "重试（上游固定）",
        "pinnedAccountId": "固定 account_id",
        "retryNotes": "重试说明",
        "requestBody": "请求体",
        "confirmRetry": "确认重试",
        "retrySuccess": "重试成功",
        "retryFailed": "重试失败",
        "retryHint": "重试将使用相同的请求参数重新发送请求",
        "retryClientHint": "使用客户端重试（不固定账号）",
        "retryUpstreamHint": "使用上游固定重试（固定到错误的账号）",
        "pinnedAccountIdHint": "（自动从错误日志获取）",
        "retryNote1": "重试会使用相同的请求体和参数",
        "retryNote2": "如果原请求失败是因为账号问题，固定重试可能仍会失败",
        "retryNote3": "客户端重试会重新选择账号",
        "retryNote4": "对不可重试的错误可以强制重试，但不推荐",
        "confirmRetryMessage": "确认要重试该请求吗？",
        "confirmRetryHint": "将使用相同的请求参数重新发送",
        "forceRetry": "我已确认并理解强制重试风险",
        "forceRetryHint": "此错误类型通常不可通过重试解决；如仍需重试请勾选确认",
        "forceRetryNeedAck": "请先勾选确认再强制重试",
        "viewRetries": "重试历史",
        "retryHistory": "重试历史",
        "tabRetries": "重试历史",
        "retrySummary": "重试摘要",
        "responseHintSucceeded": "展示重试成功的 response_preview（#{id}）",
        "responseHintFallback": "没有成功的重试结果，展示存储的 error_body",
        "suggestUpstreamResolved": "✓ 上游错误已通过重试解决，无需人工介入"
      },
      "settings": {
        "ignoreInvalidApiKeyErrors": "忽略无效 API Key 错误",
        "ignoreInvalidApiKeyErrorsHint": "启用后，无效或缺失 API Key 的错误（INVALID_API_KEY、API_KEY_REQUIRED）将不会写入错误日志。"
      }
    },
    "referral": {
      "title": "推荐设置",
      "description": "配置推荐奖励系统参数",
      "enabled": "启用推荐系统",
      "enabledDesc": "开启后用户可以通过推荐链接邀请好友注册",
      "maxPerUser": "每用户最大推荐数",
      "maxPerUserHint": "0 表示不限制",
      "referrerRewards": "推荐人奖励",
      "refereeRewards": "被推荐人奖励",
      "balanceReward": "余额奖励",
      "groupId": "订阅分组",
      "groupIdHint": "选择\"无\"表示不发放订阅奖励",
      "noGroup": "无（不发放订阅奖励）",
      "subscriptionDays": "订阅天数",
      "saved": "推荐设置已保存",
      "saveFailed": "保存推荐设置失败",
      "loadFailed": "加载推荐设置失败"
    },
    "settings": {
      "tabs": {
        "client": "客户端",
        "data": "Sora 存储"
      },
      "github": {
        "title": "GitHub 登录",
        "description": "配置 GitHub OAuth，用于 Sub2API 用户登录",
        "enable": "启用 GitHub 登录",
        "enableHint": "在登录/注册页面显示 GitHub 登录入口",
        "clientId": "Client ID",
        "clientIdPlaceholder": "Iv1.1234567890abcdef",
        "clientIdHint": "从 GitHub Developer Settings 获取",
        "clientSecret": "Client Secret",
        "clientSecretPlaceholder": "********",
        "clientSecretHint": "用于后端交换 token（请保密）",
        "clientSecretConfiguredPlaceholder": "********",
        "clientSecretConfiguredHint": "密钥已配置，留空以保留当前值。",
        "redirectUrl": "回调地址（Redirect URL）",
        "redirectUrlPlaceholder": "https://your-domain.com/api/v1/auth/oauth/github/callback",
        "redirectUrlHint": "需与 GitHub OAuth App 中配置的回调地址一致（必须是 http(s) 完整 URL）",
        "quickSetCopy": "使用当前站点生成并复制",
        "redirectUrlSetAndCopied": "已使用当前站点生成回调地址并复制到剪贴板"
      },
      "site": {
        "groupStatusEnabled": "开放模型运行状态",
        "groupStatusEnabledDescription": "开启后，普通用户侧边栏会显示“模型运行状态”菜单，并开放对应用户接口。",
        "communityQRCodePlaceholder": "粘贴二维码图片的 base64 或 URL",
        "communityQRCode": "交流群二维码",
        "uploadQRCode": "上传二维码",
        "qrCodeHint": "上传交流群二维码图片，最大 500KB。上传后顶部导航栏将展示加入交流群入口。",
        "communityGroupURL": "交流群链接",
        "communityGroupURLPlaceholder": "https://t.me/example",
        "communityGroupURLHint": "可选，填写后二维码下方会显示加入链接。必须是完整的 http(s) 链接。"
      },
      "purchase": {
        "openMode": "打开方式",
        "openModeIframe": "内嵌模式（iframe）",
        "openModeNewWindow": "新窗口",
        "openModeHint": "选择充值/订单页面的打开方式"
      },
      "clientDownloads": {
        "title": "客户端下载",
        "description": "填写 Windows 和 macOS 桌面客户端的公开下载链接，默认首页会在已配置链接时展示下载入口。全部留空时，首页将隐藏客户端相关内容，主按钮显示「开始接入API」。",
        "windowsUrl": "Windows 下载链接",
        "windowsUrlPlaceholder": "https://downloads.example.com/sub2api-windows.exe",
        "windowsUrlHint": "留空则隐藏 Windows 下载按钮。",
        "macosUrl": "macOS 下载链接",
        "macosUrlPlaceholder": "https://downloads.example.com/sub2api-macos.dmg",
        "macosUrlHint": "留空则隐藏 macOS 下载按钮。",
        "publicHint": "请使用对象存储、CDN 或发布平台提供的公开 http(s) 链接。自定义首页内容仍完全由你填写的 HTML 或 URL 控制。"
      },
      "changelog": {
        "title": "更新日志",
        "description": "管理客户端版本更新日志条目，启用后将在 /changelog 页面展示。",
        "addEntry": "新增日志",
        "version": "版本号",
        "versionPlaceholder": "如 1.0.0",
        "publishedAt": "发布日期",
        "titleLabel": "标题",
        "titlePlaceholder": "本次更新标题",
        "items": "更新点",
        "itemPlaceholder": "支持 Markdown 语法",
        "addItem": "添加更新点",
        "enabled": "启用",
        "delete": "删除",
        "deleteConfirm": "确定删除这条日志吗？",
        "emptyHint": "暂无日志条目，点击「新增日志」添加。",
        "preview": "预览",
        "edit": "编辑",
        "moveUp": "上移",
        "moveDown": "下移"
      }
    }
  },
  "modelCatalog": {
    "title": "模型广场",
    "description": "查看你当前真实可调用的分组模型卡片、官方参考价与实际扣费价对比",
    "caption": "分组 x 模型定价卡",
    "intro": "每张卡代表一个你当前可用的“分组 + 模型”。页面聚焦官方参考价与当前展示价的直观对比，默认按展示价从低到高排列。",
    "lastUpdated": "最近更新",
    "neverUpdated": "尚未加载",
    "paymentNoticeTitle": "实付价未启用",
    "paymentNoticeDescription": "未获取到支付换算配置，当前仅展示官方参考价和美元余额扣费价。",
    "filters": {
      "search": "搜索",
      "searchPlaceholder": "搜索模型名、分组名或平台",
      "platform": "平台",
      "allPlatforms": "全部平台",
      "billingMode": "计费模式",
      "allBillingModes": "全部模式",
      "sortBy": "排序"
    },
    "sorting": {
      "effectivePriceAsc": "按展示价升序",
      "modelAsc": "按模型名称"
    },
    "groupTabs": {
      "title": "按分组切换",
      "description": "优先通过分组标签切换查看，搜索和其他筛选作为辅助。",
      "allGroups": "全部分组",
      "currentGroup": "当前分组：{group}"
    },
    "filterResult": "当前显示 {visible} / {total} 张卡",
    "priceBasis": "官方价 vs 实付价（未启用时回退余额价）",
    "cnyRateReady": "实付换算已接入 ¥{rate} / $1 余额",
    "loadFailedTitle": "模型广场加载失败",
    "loadFailedDescription": "暂时无法获取模型卡片，请稍后刷新重试。",
    "emptyTitle": "没有匹配的模型卡",
    "emptyDescription": "试试放宽筛选条件，或者切换其他分组查看。",
    "billingMode": {
      "token": "Token",
      "perRequest": "按次",
      "image": "按张"
    },
    "rateSource": {
      "groupDefault": "分组默认倍率",
      "userOverride": "用户专属倍率"
    },
    "referenceSource": {
      "litellm": "LiteLLM 参考价",
      "fallback": "Fallback 参考价",
      "none": "无参考价"
    },
    "groupRateLabel": "倍率 {rate}",
    "peerGroupsLabel": "同模型共 {count} 个分组",
    "primaryPrice": "主展示价格",
    "priceLabels": {
      "input": "输入",
      "output": "输出",
      "cacheWrite": "缓存写入",
      "cacheRead": "缓存读取",
      "perRequest": "每次请求",
      "perImage": "每张图片"
    },
    "units": {
      "perMillionTokens": "每百万 Tokens",
      "perRequest": "每次请求",
      "perImage": "每张图片"
    },
    "priceColumns": {
      "official": "官方价",
      "balance": "余额价",
      "cash": "实付价"
    },
    "capabilities": {
      "promptCaching": "支持提示缓存",
      "longContext": "长上下文阈值 {threshold}",
      "tieredPricing": "{count} 档区间价",
      "userRateOverride": "专属倍率"
    },
    "expandDetails": "展开分组与区间详情",
    "collapseDetails": "收起详情",
    "intervalSectionTitle": "渠道区间价",
    "intervalDefaultLabel": "默认档位",
    "otherGroupsTitle": "同模型其他可用分组",
    "peerDisplayedPrice": "该分组展示价"
  },
  "announcements": {
    "newAnnouncement": "新公告"
  }
} as const
