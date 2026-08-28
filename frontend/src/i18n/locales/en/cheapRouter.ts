// CheapRouter 落地页文案叠加层（home.* 含 privacy/terms），
// 由 cheap-router 分支原单体 en.ts 抽取，deepMerge 在最后生效。
export default {
  "home": {
    "viewOnGithub": "View on GitHub",
    "viewDocs": "View Documentation",
    "docs": "Docs",
    "switchToLight": "Switch to Light Mode",
    "switchToDark": "Switch to Dark Mode",
    "dashboard": "Dashboard",
    "login": "Login",
    "getStarted": "Get Started",
    "goToDashboard": "Go to Dashboard",
    "loginConsole": "Sign in to dashboard",
    "headerTagline": "One-click Claude Code / Codex",
    "heroSubtitle": "One-click setup, pay-as-you-go top-up, visible usage",
    "heroDescription": "Run Claude Code and Codex on the same account — fast onboarding, friendly pricing, and clear failure reasons.",
    "navModels": "Models",
    "navPricing": "Pricing",
    "navChangelog": "Changelog",
    "hero": {
      "badge": "One-click Claude Code / Codex",
      "titleLeadPrimary": "Claude Code",
      "titleLeadSecondary": "and Codex,",
      "titleAccent": "one workspace",
      "titleTail": "is enough",
      "description": "One-click setup, pay-as-you-go top-up, and clear visibility into every charge. Claude Code, Codex, and GPT-5.5 share one account, one balance, and one usage panel — no more juggling two configs, two addresses, and two bills. After signing up on the desktop app, Claude Code and Codex are configured for you automatically — no more copy-pasting JSON configs.",
      "primaryNote": "One-click setup · easy top-up · clear spend · friendly pricing",
      "downloadPrimary": "Download now",
      "connectApi": "Use the API",
      "startApi": "Start with the API",
      "badgeDiscount": "vs official API",
      "tags": {
        "coding": "Claude Code",
        "agent": "Codex",
        "tools": "Multi-account · Multi-model"
      },
      "stats": {
        "setupValue": "1 step",
        "savings": "Local tools configured automatically",
        "routesValue": "2 setups",
        "models": "Claude / Codex managed separately",
        "workspaceValue": "1 place",
        "minCost": "Balance, usage, and failure reasons all here"
      },
      "perToken": "reference price",
      "panel": {
        "title": "One entry point for daily coding",
        "auto": "Auto Rotating",
        "requestLabel": "request",
        "modelLabel": "models",
        "routeLabel": "line",
        "billLabel": "billing",
        "scenarios": {
          "completions": {
            "title": "One entry for premium models, usage control, and auto-switching",
            "subtitle": "Keep the familiar OpenAI chat style while moving model access, billing, and switching into one place.",
            "route": "Multi-account auto-switching",
            "billing": "Pay for what you use, traceable"
          },
          "responses": {
            "title": "Not just chat. Tool calls and long tasks fit here too.",
            "subtitle": "Built for tool-enabled, long-running workloads without splitting different styles into separate services.",
            "route": "Tool calls + long-running tasks",
            "billing": "Pay for what you use, traceable"
          },
          "messages": {
            "title": "Claude conversations stay inside the same entry",
            "subtitle": "Keep Claude-native patterns while reusing unified usage limits, switching, and billing.",
            "route": "Claude-native + unified entry",
            "billing": "Pay for what you use, traceable"
          }
        }
      }
    },
    "pricing": {
      "title": "Metered Pricing",
      "subtitle": "vs Official Claude Code / Codex API Services",
      "highlight": "Friendlier than the official API",
      "description": "Run premium models at a friendlier price than the official API, with one entry point for both Claude Code and Codex. Balance never expires — pay only for what you use.",
      "badge": "Premium model access without the premium subscription",
      "barTitle": "Pay-as-you-go, balance never expires",
      "points": {
        "metered": "Token-level metering — pay only for what runs",
        "routing": "Multi-account auto-switching — a single key failing does not break the whole route",
        "visibility": "5h / 1d / 7d usage windows visible"
      }
    },
    "proofStrip": {
      "overline": "Platform capabilities"
    },
    "clientShowcase": {
      "badge": "Client preview",
      "title": "Relay gateway × agent workbench, deeply unified in one desktop client",
      "description": "Sign in and you are routed — the gateway key is written into every CLI's global config automatically. Balance, group rate multipliers, and uptime sit right next to your tasks, so you can hop to a cheaper, steadier route while a task is still running.",
      "pills": {
        "autoRoute": "Sign in, routed",
        "groupSwitch": "One-click group switch (rate · uptime)",
        "liveBalance": "Live balance",
        "cliInstall": "Missing CLI? One-click install"
      },
      "advantages": {
        "tiny": {
          "title": "An installer of just tens of MB",
          "body": "A single native binary — no Electron shell"
        },
        "native": {
          "title": "Native-grade smoothness",
          "body": "Rust + GPUI rendering — zero lag on scroll and input"
        },
        "ready": {
          "title": "Works out of the box",
          "body": "Sign in and you are routed; missing Node or a CLI is fixed in one click"
        }
      },
      "apiOnly": {
        "title": "Just want the raw API?",
        "body": "Register for a key and call the OpenAI/Anthropic-compatible endpoints directly — works with third-party clients of every kind.",
        "dashboardCta": "Get an API key",
        "docsCta": "Read the docs"
      },
      "cta": "Get client updates",
      "ctaNote": "Register to get desktop client updates first.",
      "downloadCta": "Download {platform}",
      "downloadNote": "Desktop client downloads are available now."
    },
    "clientWorkflow": {
      "ariaLabel": "CheapRouter desktop client demo: switching routing groups with one click while a task is running",
      "working": "工作中 · {seconds} 秒",
      "balanceBefore": "$999990.44",
      "balanceAfter": "$999990.41",
      "sidebar": {
        "newTask": "新建任务",
        "search": "搜索",
        "today": "今天",
        "taskTitle": "在吗",
        "project": "amadeus-system",
        "email": "huzw1995{'@'}163.com"
      },
      "labels": {
        "read": "读取",
        "tool": "工具",
        "thinking": "思考"
      },
      "rows": {
        "r1": "main.ts",
        "r2": "List `D:\\Projects\\amadeus-system\\src\\views\\chat\\components`",
        "r3": "constants.ts",
        "r4": "思考用时 1 秒",
        "r5": "chat.ts",
        "r6": "auth.ts",
        "r7": "user.ts",
        "r8": "database.ts",
        "r9": "changelog.ts",
        "r10": "List `D:\\Projects\\amadeus-system\\src\\views\\chat\\hooks`",
        "r11": "index.vue",
        "r12": "auth.ts"
      },
      "composer": {
        "placeholder": "做什么都可以…",
        "model": "Grok 4.6",
        "effort": "High",
        "access": "完全访问",
        "build": "构建",
        "stop": "停止"
      },
      "statusBar": {
        "project": "amadeus-system",
        "local": "本地",
        "branch": "my_feature"
      },
      "menu": {
        "balance": "余额 {amount}",
        "topUp": "充值",
        "claudeGroup": "Claude Code 分组",
        "claudeValue": "Claude Sale",
        "codexGroup": "Codex 分组",
        "codexValueBefore": "Codex",
        "codexValueAfter": "Codex Sale",
        "grokGroup": "Grok 分组",
        "grokValue": "Grok",
        "modelPlaza": "模型广场",
        "usage": "使用记录",
        "logout": "退出登录",
        "submenu": {
          "default": "账号默认",
          "codexName": "Codex",
          "codexMeta": "×0.29 · 96.2%",
          "saleName": "Codex Sale",
          "saleMeta": "×0.19 · 86.4%",
          "welfareName": "Codex 福利分组",
          "welfareMeta": "×0.09 · 69.3%"
        }
      },
      "toast": "已切换到 Codex Sale，对新启动的任务生效"
    },
    "download": {
      "badge": "Desktop client",
      "title": "Download the CheapRouter desktop client",
      "description": "Install the desktop app to configure Claude Code and Codex automatically, reuse existing local settings safely, and keep usage status in one place.",
      "privacyCode": "Auto-writes local config files without overwriting existing settings",
      "privacyKey": "Desktop, Web, and mobile share the same usage and failure info",
      "cta": "Download",
      "recommended": "Recommended for this device",
      "platforms": {
        "mac": {
          "sub": "Apple Silicon & Intel"
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
      "overline": "Why CheapRouter",
      "title": "The boring parts of running Claude Code and Codex, made easy",
      "description": "Not another IDE, and not just another relay service. CheapRouter pulls together the scattered chores — sign up, top up, configure local tools, watch usage, debug failures — into one smooth path.",
      "items": {
        "economics": {
          "eyebrow": "01 / Fast onboarding",
          "title": "Local config, written for you",
          "description": "After signing up on the desktop app, Claude Code and Codex are configured for you automatically — no more copy-pasting JSON configs, without overwriting your existing settings."
        },
        "reliability": {
          "eyebrow": "02 / Spend you can see",
          "title": "One balance panel for both tools",
          "description": "How much Claude and Codex have used, how much credit is left, what the current speed limit looks like — all in one place, no more switching between platforms."
        },
        "control": {
          "eyebrow": "03 / All in one",
          "title": "SLA-backed service you can rely on",
          "description": "Service availability, response speed, and pricing strategy are managed uniformly across all lines. When issues occur, you are not left guessing — traffic automatically switches to healthy routes, or you get clear recovery guidance."
        }
      }
    },
    "comparison": {
      "overline": "A different angle",
      "title": "You are probably already using one of these",
      "description": "",
      "headers": {
        "feature": "How you do it today",
        "official": "Where it falls short",
        "us": "How CheapRouter handles it"
      },
      "items": {
        "pricing": {
          "feature": "Official accounts + manual config",
          "official": "One account per tool — multiple accounts, multiple configs, multiple bills, and you maintain everything by hand",
          "us": "One account, both tools configured automatically, settings can be reused, balance shared across tools"
        },
        "models": {
          "feature": "Local switcher scripts",
          "official": "Can swap addresses, but cannot tell you the remaining balance, current credit, or speed-limit state",
          "us": "After a switch, balance, credit, recent usage, and provider health are all visible"
        },
        "stability": {
          "feature": "Workflow",
          "official": "Cursor and Windsurf require uploading your code to their servers",
          "us": "Unlike Cursor or Windsurf, we respect your local Claude Code and Codex workflow — MCP, Skill, and Workflow require no extra configuration"
        }
      }
    },
    "pricingTable": {
      "overline": "Pricing advantage",
      "title": "Mainstream coding models at friendlier prices than the official API",
      "description": "Premium coding models — Claude Code, Codex, and GPT-5.5 — at much better prices than official API rates. One account, one balance, pay for what you use, balance never expires.",
      "badge": "Goal",
      "badgeValue": "Lower total cost",
      "badgeSavings": "Max savings",
      "badgeSavingsValue": "{percent}% off",
      "currencyNote": "Prices shown converted at 1 USD ≈ CNY {rate}; billing is settled in USD.",
      "table": {
        "model": "Model",
        "group": "Group",
        "input": "Input / 1M",
        "output": "Output / 1M",
        "cacheWrite": "Cache write / 1M",
        "cacheRead": "Cache read / 1M",
        "discountHeader": "vs official",
        "discount": "{percent}% off",
        "perRequest": "{price} / request",
        "perImage": "{price} / image",
        "modelCount": "{count} models",
        "expand": "Show all ({count} more)",
        "collapse": "Show less"
      },
      "cards": {
        "claude": {
          "tag": "Claude family",
          "title": "For heavy Claude Code work",
          "description": "Full coverage of Claude Sonnet 4.6 / Opus 4.7 / Haiku 4.5. Daily coding, refactors, and reviews on Claude Code, with each line billed separately."
        },
        "codex": {
          "tag": "Codex / GPT family",
          "title": "Codex and GPT-5.5, both here",
          "description": "New GPT-5.5 available alongside GPT-5.4 and GPT-5.3 Codex. Local Codex is configured automatically — no manual edits needed."
        },
        "compatible": {
          "tag": "OpenAI-compatible · others",
          "title": "Other compatible models, same gateway",
          "description": "Gemini, GLM, Qwen and more compatible models share the same entry; pricing follows what you see in your dashboard, with clear failure messages too."
        }
      },
      "note": "No \"absolute lowest price\" promise here. CheapRouter focuses on total cost: one place to top up, pay for what you use, usage tracked per line, visible credit and speed limits, and clear failure reasons. Discounts vary by provider and model — final billing follows the price shown in your dashboard."
    },
    "providers": {
      "title": "Premium models and local coding workflows under one service layer",
      "description": "Built around high-frequency coding agents rather than a generic model catalog, while keeping Claude Code, Codex, and OpenAI-compatible request shapes.",
      "supported": "Supported",
      "claude": "Claude",
      "claudeCode": "Claude Code",
      "gpt": "GPT",
      "codex": "Codex",
      "gemini": "Gemini",
      "openaiCompatible": "OpenAI-Compatible"
    },
    "trust": {
      "overline": "Run with confidence",
      "title": "See where the money goes and why it cannot run",
      "description": "The biggest pain with relay services is opacity — you do not know how much balance is left, and you do not know what broke. CheapRouter makes those two things explicit so you do not have to guess.",
      "cards": {
        "gateway": {
          "title": "Each line billed separately",
          "description": "Claude Code and Codex use their own settings without overwriting each other. Per-line current spend, recent usage, and remaining credit are tracked separately."
        },
        "resilience": {
          "title": "Provider issues, surfaced early",
          "description": "Each provider shows availability, response speed, and pricing — so when something breaks you know which line is affected and can switch manually or wait for auto-recovery."
        },
        "visibility": {
          "title": "Errors come with reasons, not just numbers",
          "description": "Out of balance, credit exhausted, speed-limited, config mismatch, line mismatch, provider down, or protocol incompatibility — each of these seven common failures gets a specific message so you are not staring at a bare error code."
        }
      },
      "trackers": {
        "routing": "Pay-as-you-go",
        "billing": "Balance never expires",
        "visibility": "Multi-account switching"
      }
    },
    "cta": {
      "eyebrow": "GET STARTED",
      "title": "Claude Code and Codex, set up in one go",
      "description": "Sign up, top up, and let CheapRouter configure your local tools — three steps and you are running. The full Claude family, Codex, and GPT-5.5 share the same balance, and the dashboard shows exactly which line every charge is billed against.",
      "button": "Start setup",
      "stat": "One-click setup · pay-as-you-go · visible usage · friendly pricing"
    },
    "footer": {
      "allRightsReserved": "All rights reserved.",
      "privacy": "Privacy Policy",
      "terms": "Terms of Service"
    },
    "privacy": {
      "title": "Privacy Policy",
      "backHome": "Back to home",
      "lastUpdated": "Last updated: April 2026",
      "intro": "This Privacy Policy explains how CheapRouter (\"we\", \"us\") collects, uses, and protects your information when you use our platform. We are committed to protecting your privacy, especially your code and prompt content.",
      "pledge": {
        "title": "Your code and prompts are never stored or used for training",
        "body": "All code, prompts, and conversation content you send through CheapRouter is forwarded directly to the official upstream API. We do not store, log, or analyze request content, and we never use it for model training or any other purpose."
      },
      "sections": {
        "collection": {
          "title": "1. Information We Collect",
          "p1": "We collect only the minimum information necessary to provide our service:",
          "i1": "Account information: email address provided at registration",
          "i2": "Usage records: token counts, request counts, model type (no request content)",
          "i3": "Payment information: top-up amounts and transaction records (no card numbers or sensitive financial data)",
          "i4": "Log information: IP address and request timestamps for security and troubleshooting"
        },
        "use": {
          "title": "2. How We Use Your Information",
          "p1": "Collected information is used only for the following purposes:",
          "i1": "Providing and maintaining the API proxy service",
          "i2": "Calculating token usage and generating billing records",
          "i3": "Sending service notifications (e.g. low balance alerts)",
          "i4": "Ensuring account security and preventing abuse"
        },
        "code": {
          "title": "3. Code & Prompt Protection",
          "p1": "All request content you send via the API, including code, prompts, and conversation messages, is forwarded transparently to the official Anthropic and OpenAI APIs. CheapRouter does not persist or store request content.",
          "p2": "We will never use your code or prompts to train, fine-tune, or evaluate any machine learning model, and we will not share this content with third parties."
        },
        "apikey": {
          "title": "4. API Key Security",
          "p1": "API keys you generate on this platform are stored encrypted at rest. Upstream API credentials used to call model providers are managed centrally by the platform and are never exposed to users. Keep your CheapRouter API key safe. If you suspect compromise, reset it immediately in the dashboard."
        },
        "third": {
          "title": "5. Third-Party Services",
          "p1": "We forward API requests to the following third-party providers, each with their own privacy policies:",
          "i1": "Anthropic (Claude model family)",
          "i2": "OpenAI (GPT / Codex model family)"
        },
        "retention": {
          "title": "6. Data Retention",
          "p1": "Account information is retained until you delete your account. Usage records (token counts and timestamps, no content) are retained for 90 days for billing reconciliation. You may request account deletion and removal of associated data at any time from account settings."
        },
        "rights": {
          "title": "7. Your Rights",
          "p1": "You have the following rights over your personal data:",
          "i1": "Access: view the data we hold about you",
          "i2": "Correction: update inaccurate account information",
          "i3": "Deletion: request account closure and removal of your data"
        },
        "changes": {
          "title": "8. Policy Changes",
          "p1": "If we make material changes to this policy, we will notify you via in-app notice or email in advance. Continued use of the service constitutes acceptance of the updated policy."
        }
      },
      "contact": "If you have any questions about this Privacy Policy, please contact us through the support channel in the platform."
    },
    "terms": {
      "title": "Terms of Service",
      "lastUpdated": "Last updated: April 2026",
      "intro": "Welcome to CheapRouter. Please read these Terms of Service carefully before using our platform. By registering or using the service, you agree to be bound by these terms.",
      "sections": {
        "eligibility": {
          "title": "1. Eligibility",
          "p1": "This service is available to individuals with full legal capacity and legally registered business entities. Users under 18 must have parental or guardian consent. By using this service you represent and warrant that you have the authority to accept these terms."
        },
        "account": {
          "title": "2. Account Responsibilities",
          "p1": "You are fully responsible for the security of your account and all activity that occurs under it:",
          "i1": "Keep your login credentials and API keys safe. Do not share them with others",
          "i2": "Notify us immediately if you detect unauthorized access or account anomalies",
          "i3": "Transferring, selling, or sharing accounts is prohibited"
        },
        "service": {
          "title": "3. Service Description",
          "p1": "CheapRouter is an AI API aggregation proxy that forwards your requests to official providers such as Anthropic and OpenAI. We are not responsible for the availability, response quality, or content of upstream providers.",
          "p2": "We reserve the right to modify, suspend, or terminate the service with advance notice. Service interruptions caused by upstream provider outages, maintenance, or force majeure events are outside our responsibility."
        },
        "billing": {
          "title": "4. Billing & Refunds",
          "p1": "This platform operates on a prepaid credit model:",
          "i1": "Credits are non-refundable once purchased. Top up only what you need",
          "i2": "Token consumption is deducted in real time based on platform records",
          "i3": "Abnormal charges caused by platform errors may be investigated and compensated upon request"
        },
        "prohibited": {
          "title": "5. Prohibited Uses",
          "p1": "When using this service, you must not:",
          "i1": "Use the platform to generate illegal, harmful, discriminatory, or infringing content",
          "i2": "Attempt to reverse-engineer, bypass, or abuse the platform rate limits or security mechanisms",
          "i3": "Resell, distribute, or use API keys for unauthorized commercial proxy services",
          "i4": "Violate the usage policies of upstream model providers (Anthropic, OpenAI, etc.)"
        },
        "ip": {
          "title": "6. Intellectual Property",
          "p1": "The platform interface, codebase, and branding are the intellectual property of CheapRouter. Ownership of content generated via the API is governed by the upstream provider policies. We make no claim over generated output."
        },
        "disclaimer": {
          "title": "7. Disclaimer of Warranties",
          "p1": "This service is provided \"as is\" without any express or implied warranties. To the maximum extent permitted by law, CheapRouter is not liable for any indirect, incidental, special, or consequential damages, including but not limited to lost profits, data loss, or business interruption."
        },
        "termination": {
          "title": "8. Account Termination",
          "p1": "We reserve the right to suspend or terminate accounts that violate these terms, without prior notice. You may also delete your account at any time from account settings. Remaining balance is non-refundable upon termination."
        },
        "changes": {
          "title": "9. Changes to Terms",
          "p1": "We may update these terms from time to time. Material changes will be communicated in advance via in-app notice or email. Continued use of the service constitutes acceptance of the updated terms."
        }
      },
      "contact": "If you have any questions about these Terms of Service, please contact us through the support channel in the platform."
    }
  }
}
