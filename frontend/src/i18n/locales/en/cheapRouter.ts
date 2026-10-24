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
    "navPricing": "Models & pricing",
    "navChangelog": "Changelog",
    "landing": {
      "hero": {
        "releaseBadge": "v{version} is out",
        "releaseCta": "See what changed",
        "tagline": "Smart models, checked nonstop.",
        "client": {
          "label": "Our Own Agent desktop client",
          "title": "Every agent, one workspace, every API.",
          "subtitle": "Works out of the box — no setup at all, with a built-in agent. Native Rust, light but complete: plan review, parallel sub-agents, agent teams and image generation, with Claude Code and Codex in the same window."
        },
        "api": {
          "label": "A quality-checked AI relay",
          "title": "One API key for all.",
          "subtitle": "OpenAI- and Anthropic-compatible endpoints for Claude, GPT and Grok, plus DeepSeek, GLM, Kimi and other Chinese models. Fingerprint checks confirm the model's identity, and anything off turns red and pages us at once."
        },
        "downloadFor": "Download for {platform}",
        "copyInstall": "Copy the {platform} install command",
        "switchPlatform": "Get the {platform} version",
        "useApi": "Use the API directly",
        "startApi": "Start with the API",
        "viewDocs": "Read the docs",
        "installHint": "Paste it into a terminal to install",
        "terminal": {
          "title": "Terminal",
          "claudeComment": "# Claude Code",
          "openaiComment": "# Codex · OpenAI SDK · any compatible client",
          "caption": "Swap the base URL and key — your tools and code stay exactly as they are."
        }
      },
      "features": {
        "client": {
          "titleLead": "Agents need a system,",
          "titleTail": "not a pile of config files.",
          "subtitle": "Sign in once and Claude Code, Codex and the rest are wired up — any already on your own account is asked about first; plans, teams, balance and usage live in the same window."
        },
        "api": {
          "titleLead": "Models under inspection,",
          "titleTail": "one key to reach them.",
          "subtitle": "Every probe and identity check is on record, and trouble turns red before it pages us; Claude, GPT, Grok and Chinese models such as DeepSeek, GLM and Kimi share one balance and one set of usage numbers."
        },
        "columns": {
          "quality": "Quality-checked relay",
          "client": "Our own agent client"
        },
        "cards": {
          "quality": {
            "title": "Model identity checks",
            "body": "GPT routes are checked on a schedule with Sol and Astra fingerprints; repeated mismatches mark the route as failing"
          },
          "probe": {
            "title": "Live route probes",
            "body": "Availability and first-token latency are measured nonstop; trouble turns red and pages on-call right away"
          },
          "builtinAgent": {
            "title": "Built-in agent",
            "body": "Ready with the app — no Node, no CLI. It reads and draws pictures, drives the in-app browser on Windows for testing, and runs sub-agents in parallel"
          },
          "agentTeams": {
            "title": "Agent teams · plan review",
            "body": "/agent-teams turns a goal into a team, each member on its own model and every task checked before it counts; plans arrive in full, compared version by version, and nothing runs until you approve"
          },
          "allAgents": {
            "title": "Every agent, one window",
            "body": "Claude Code, Codex, Grok, OpenCode and Pi are wired up at sign-in; any already on your own account or relay is asked about first and left untouched if you keep it. Amp, Cursor, Kimi Code and more run here too"
          },
          "compatible": {
            "title": "Native protocols",
            "body": "OpenAI Chat / Responses and Anthropic Messages work as they are"
          },
          "failover": {
            "title": "Automatic failover",
            "body": "When one route fails the next takes over, without dropping the request"
          },
          "metered": {
            "title": "Pay as you go",
            "body": "Pay only for what runs, no minimum spend, balance never expires"
          },
          "invoice": {
            "title": "Invoices available",
            "body": "Invoices for every top-up, so team expenses stay simple"
          }
        }
      },
      "models": {
        "overline": "Many models · one switch",
        "titleLead": "Not one model.",
        "titleTail": "The right one for every step.",
        "clientSubtitle": "Switch between the built-in agent, Claude Code, Codex and Grok in one client — every model says what it's for and your own endpoints get their own section — while your project, files and context stay put.",
        "apiSubtitle": "Claude, GPT, Grok and Chinese models such as DeepSeek, GLM and Kimi share one key and one balance; changing models means changing one parameter.",
        "cta": "See model pricing"
      },
      "closing": {
        "overline": "Get started",
        "clientTitleLead": "Download {siteName},",
        "clientTitleTail": "run your first task in minutes.",
        "apiTitleLead": "Sign up for a key,",
        "apiTitleTail": "send your first request in minutes.",
        "note": "Routes checked nonstop · Pay as you go · Balance never expires · Invoices available"
      },
      "footer": {
        "clientTagline": "A quality-checked AI relay × agent desktop client",
        "apiTagline": "A quality-checked AI model gateway",
        "product": "Product",
        "support": "Support",
        "download": "Download",
        "pricing": "Pricing",
        "changelog": "Changelog",
        "docs": "Docs",
        "console": "Dashboard"
      }
    },
    "hero": {
      "badge": "One-click Claude Code / Codex",
      "titleLeadPrimary": "A price-competitive AI relay",
      "titleLeadSecondary": "× Agent desktop client",
      "description": "Deeply unified desktop agent client for Claude Code · Codex · Grok · Pi — sign in and you are routed, with balance, group rates, and uptime at a glance.",
      "primaryNote": "One desktop client for all your agent CLIs",
      "downloadPrimary": "Download client",
      "installPrimary": "Copy install command",
      "connectApi": "Use the API",
      "startApi": "Start with the API",
      "badgeDiscount": "vs official API",
      "tags": {
        "coding": "Claude Code",
        "agent": "Codex",
        "tools": "Grok · Pi · More",
        "more": "More"
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
    "clientWorkflow": {
      "ariaLabel": "{siteName} desktop client demo in three parts: the built-in agent hands over a plan, then after approval runs two sub-agents in parallel and edits the code; /agent-teams starts a three-member team whose tasks finish in dependency order, each passing a check; finally the Images page draws a picture",
      "chapters": {
        "label": "Demo chapters",
        "plan": "Plan → run",
        "team": "Agent team",
        "image": "Images",
        "pause": "Pause the demo",
        "play": "Play the demo"
      },
      "sidebar": {
        "newTask": "New Task",
        "search": "Search",
        "images": "Images",
        "modelStatus": "Model status",
        "viewByProject": "By project",
        "viewTimeline": "Timeline",
        "projects": "Projects",
        "tasks": "Tasks",
        "taskTitle": "Return to the page after login",
        "teamTaskTitle": "Split the settings page",
        "teamTime": "1h",
        "elapsed": "{seconds}s",
        "justNow": "just now",
        "olderTask": "Add a client download button",
        "olderTime": "3h",
        "email": "admin{'@'}cheaprouter.cc"
      },
      "overview": {
        "headlineLead": "What should we build in ",
        "headlineTail": "?",
        "tabOverview": "Overview",
        "tabModels": "Models",
        "rangeAll": "All",
        "range30d": "30d",
        "range7d": "7d",
        "sessions": "Sessions",
        "messages": "Messages",
        "activeDays": "Active days",
        "peakHour": "Peak hour",
        "peakHourValue": "3 PM",
        "favoriteModel": "Favorite model",
        "longestStreak": "Longest streak",
        "streakValue": "12 days",
        "heatCaption": "3,642 messages in the last 6 months",
        "less": "Less",
        "more": "More"
      },
      "transcript": {
        "prompt": "After signing in, send people back to the page they came from instead of the home page",
        "explored": "Explored · 1 search, 2 files",
        "thought": "Thought for 3s",
        "handedOver": "Handed over the plan",
        "subagent": "SubAgent",
        "subagentFind": "Find every route that sends people to sign-in",
        "subagentGuard": "Read the redirect logic in the route guard",
        "edited": "Edited",
        "running": "Running",
        "working": "Working for {seconds}s",
        "workedFor": "Worked for {seconds}s",
        "reply": "Done: the page you came from is remembered before sign-in and you land back on it afterwards, falling back to the dashboard. All tests pass.",
        "changedFiles": "Changed {count} files",
        "review": "Review",
        "undo": "Undo"
      },
      "status": {
        "tokens": "{count} tokens",
        "thinking": "Thinking…",
        "writing": "Writing…",
        "runningTool": "Running a tool…",
        "runningTools": "Running {count} tools…",
        "waitingForModel": "Waiting for the model…",
        "waitingForYou": "Waiting for you"
      },
      "composer": {
        "placeholder": "Do anything…",
        "effort": "High",
        "access": "Full access",
        "build": "Build",
        "plan": "Plan"
      },
      "footer": {
        "project": "amadeus-web",
        "local": "Local"
      },
      "picker": {
        "search": "Search models…",
        "builtinAgent": "Built-in agent",
        "vendorZhipu": "Zhipu GLM",
        "ownEndpoint": "My relay",
        "descriptions": {
          "claudeSonnet": "Balanced speed and intelligence for everyday agentic coding.",
          "claudeOpus": "Large hybrid-reasoning model for complex coding and agents.",
          "claudeFable": "Anthropic's frontier line, for the hardest coding and knowledge work."
        }
      },
      "plan": {
        "surface": "Plan",
        "readyTitle": "Finished planning",
        "cardHint": "Read the whole plan and add notes in the right panel.",
        "viewInPanel": "View plan",
        "approve": "Approve plan",
        "keepPlanning": "Keep planning",
        "keepPlanningWithNotes": "Keep planning with these notes",
        "statusPending": "Waiting for your approval",
        "statusApproved": "Approved",
        "quote": "Quote selection",
        "quoteHint": "Select part of the plan, quote it, then write what to change below.",
        "notesPlaceholder": "What should change? Enter sends the plan back, Shift+Enter starts a new line.",
        "title": "Return to the page after sign-in",
        "step1": "Put the current address in a redirect parameter before opening sign-in",
        "step2": "After sign-in, check redirect and accept only same-site paths",
        "step3": "Fall back to the dashboard when there is none",
        "step4": "Add a unit test for the route guard"
      },
      "team": {
        "surface": "Team",
        "name": "Settings split",
        "command": "split the settings page into its own module and add tests",
        "draftReply": "The team plan is ready: 3 members, 4 tasks. Approve it in the Team panel on the right to start.",
        "doneReply": "All 4 tasks are done and each passed its check; T2 failed the first review and passed after rework.",
        "phaseReview": "Awaiting review",
        "phaseRunning": "Running",
        "phaseDone": "Finished",
        "stats": "{working} of {members} members working · {done} of {tasks} tasks done · {messages} messages",
        "approve": "Approve and run",
        "revise": "Return to chat to revise",
        "discard": "Discard plan",
        "stop": "Stop team",
        "members": "Members",
        "tasks": "Tasks",
        "memberWorking": "working",
        "memberIdle": "idle",
        "memberWaiting": "not started",
        "memberTasks": "{done}/{total} tasks",
        "memberCurrent": "Working on {task}",
        "taskBlocked": "blocked",
        "taskOpen": "ready",
        "taskRunning": "running",
        "taskCompleted": "done",
        "taskRound": "round {round}",
        "effortHigh": "high",
        "effortMedium": "medium",
        "memberNames": {
          "architect": "architect",
          "builder": "builder",
          "tester": "tester"
        },
        "taskSubjects": {
          "t1": "Extract the settings module",
          "t2": "Move the form components",
          "t3": "Add unit tests",
          "t4": "Update the routes"
        }
      },
      "image": {
        "title": "Images",
        "pictureCount": "{count} pictures",
        "openFolder": "Open folder",
        "placeholder": "Describe a picture, or drop and paste images to edit them…",
        "prompt": "An orange kitten napping on a keyboard, warm tones",
        "addImages": "Add images",
        "group": "Group: automatic",
        "quality": "Quality: high",
        "count": "1 picture",
        "mode": "Text to image",
        "estimate": "About $0.04",
        "generate": "Draw",
        "drawing": "Drawing · {time}",
        "runningTask": "Runs on the gateway — closing the window loses nothing",
        "gallery": {
          "sunset": "Beach at sunset, film look",
          "mountain": "A starry camp below snowy peaks",
          "city": "Neon street on a rainy night"
        }
      }
    },
    "download": {
      "commandCopied": "Install command copied. Paste it in your terminal to run."
    },
    "comparison": {
      "overline": "A different angle",
      "title": "You are probably already using one of these",
      "headers": {
        "feature": "How you do it today",
        "official": "Where it falls short",
        "us": "How {siteName} handles it",
      },
      "items": {
        "pricing": {
          "feature": "Official accounts + manual config",
          "official": "One account per tool — multiple accounts, multiple configs, multiple bills, and you maintain everything by hand",
          "us": "One account and one key: point Claude Code, Codex and other tools at a new base URL, with one balance across them",
          "usClient": "One account sets up Claude Code, Codex and more at sign-in — any you already run on your own account can stay as it is — with one balance across tools"
        },
        "models": {
          "feature": "Local switcher scripts",
          "official": "Can swap addresses, but cannot tell you the remaining balance, current credit, or speed-limit state",
          "us": "After a switch, balance, credit, recent usage, and provider health are all visible"
        },
        "stability": {
          "feature": "Workflow",
          "official": "Cursor and Windsurf require uploading your code to their servers",
          "us": "Your code stays local; Claude Code and Codex keep using your MCP servers and Skills with no extra setup",
          "usClient": "Your code stays local and Claude Code and Codex keep your MCP servers and Skills; the built-in agent can also run a team with /agent-teams, each task checked before it counts"
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
      "note": "No \"absolute lowest price\" promise here. {siteName} focuses on total cost: one place to top up, pay for what you use, usage tracked per line, visible credit and speed limits, and clear failure reasons. Discounts vary by provider and model — final billing follows the price shown in your dashboard.",
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
    "footer": {
      "allRightsReserved": "All rights reserved.",
      "privacy": "Privacy Policy",
      "terms": "Terms of Service",
      "aup": "Acceptable Use Policy",
      "onlineSupport": "Contact support",
      "contact": "Contact"
    },
    "legal": {
      "onlineSupport": "Online support: sign in and open a support ticket; our staff reply in the ticket.",
      "onlineSupportLink": "Open a ticket",
      "contactLabel": "Support contact: ",
      "communityHint": "Scan to join our community group and reach support"
    },
    "privacy": {
      "title": "Privacy Policy",
      "backHome": "Back to home",
      "lastUpdated": "Last updated: October 2026",
      "intro": "This Privacy Policy explains how {siteName} (\"we\", \"us\") collects, uses, and protects your information when you use our platform. We are committed to protecting your privacy, especially your code and prompt content.",
      "pledge": {
        "title": "Your code and prompts are never used for training",
        "body": "All code, prompts, and conversation content you send through {siteName} is forwarded directly to the official upstream API. We never use it for model training, sell it or use it for advertising, and apart from the content safety moderation described in section 3 we do not keep request content.",
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
          "p1": "All request content you send via the API, including code, prompts, conversation messages and uploaded images, is forwarded transparently to the official Anthropic and OpenAI APIs. {siteName} does not persist request content.",
          "p2": "We will never use your code or prompts to train, fine-tune, or evaluate any machine learning model, and we will not sell this content or share it with unrelated third parties.",
          "p3": "To enforce the AIGC Acceptable Use Policy, request content is screened by automated content safety checks, which may be performed by third-party content safety services or large language models. Only when a request triggers a moderation rule do we record the categories and an excerpt of at most 240 characters of the input, used to enforce the policy and review appeals, and delete it after the retention period set by the platform (180 days by default)."
        },
        "apikey": {
          "title": "4. API Key Security",
          "p1": "API keys you generate on this platform are stored encrypted at rest. Upstream API credentials used to call model providers are managed centrally by the platform and are never exposed to users. Keep your {siteName} API key safe. If you suspect compromise, reset it immediately in the dashboard.",
        },
        "third": {
          "title": "5. Third-Party Services",
          "p1": "We forward API requests to the following third-party providers, each with their own privacy policies:",
          "i1": "Anthropic (Claude model family)",
          "i2": "OpenAI (GPT / Codex model family)",
          "i3": "Content safety services (for automated moderation; they receive only the request content being checked)"
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
      "lastUpdated": "Last updated: October 2026",
      "intro": "Welcome to {siteName}. Please read these Terms of Service carefully before using our platform. By registering or using the service, you agree to be bound by these terms.",
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
          "p1": "{siteName} is an AI API aggregation proxy offering chat, code generation and image generation (text-to-image and image editing). It forwards your requests to official providers such as Anthropic and OpenAI. We are not responsible for the availability, response quality, or content of upstream providers.",
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
          "i1": "Generate, edit, or distribute content prohibited by the AIGC Acceptable Use Policy, including the six categories of sexual / NSFW content, violence / gore, hate speech, child safety (CSAM) content, deepfakes / impersonation, and copyright / trademark infringement, or any other illegal or harmful content",
          "i2": "Attempt to reverse-engineer, bypass, or abuse the platform rate limits or security mechanisms",
          "i3": "Resell, distribute, or use API keys for unauthorized commercial proxy services",
          "i4": "Violate the usage policies of upstream model providers (Anthropic, OpenAI, etc.)",
          "p2": "The platform moderates generation requests automatically and accepts reports of prohibited content. The full list of prohibited content, how moderation works, how to report and how violations are handled are set out in the AIGC Acceptable Use Policy, which forms part of these terms.",
          "link": "Read the AIGC Acceptable Use Policy"
        },
        "ip": {
          "title": "6. Intellectual Property",
          "p1": "The platform interface, codebase, and branding are the intellectual property of {siteName}. Ownership of content generated via the API is governed by the upstream provider policies. We make no claim over generated output.",
        },
        "disclaimer": {
          "title": "7. Disclaimer of Warranties",
          "p1": "This service is provided \"as is\" without any express or implied warranties. To the maximum extent permitted by law, {siteName} is not liable for any indirect, incidental, special, or consequential damages, including but not limited to lost profits, data loss, or business interruption."
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
    },
    "aup": {
      "title": "AIGC Acceptable Use Policy",
      "lastUpdated": "Last updated: October 2026",
      "intro": "This policy covers every generation capability you use through {siteName}, including chat, code generation, text-to-image and image editing. It is part of the Terms of Service; by registering or using the service you agree to follow it. You are responsible for the prompts and materials you submit and for how you use the generated output.",
      "sections": {
        "prohibited": {
          "title": "1. Prohibited Content",
          "p1": "Whatever the purpose, you must not use the platform to generate, edit or distribute the following six categories of content, or prompt a model to produce them:",
          "nsfw": {
            "label": "Sexual / NSFW content",
            "text": "Explicit depictions of sexual acts, sexualized nudity, solicitation of sexual services, and any sexualized content involving real people."
          },
          "violence": {
            "label": "Violence / gore",
            "text": "Content that glorifies, promotes or incites violence or terrorism, specific instructions for harming others or for self-harm or suicide, and graphic gore, cruelty or abuse presented for its own sake."
          },
          "hate": {
            "label": "Hate speech",
            "text": "Content that insults, discriminates against, harasses, threatens or dehumanizes people or groups on the basis of race, ethnicity, nationality, religion, gender, sexual orientation, disability or similar characteristics."
          },
          "csam": {
            "label": "Child safety (CSAM)",
            "text": "Any content that sexualizes minors, including fictional, animated or AI-generated depictions, and any content that grooms, exploits or harms minors. Zero tolerance applies."
          },
          "deepfake": {
            "label": "Deepfakes / impersonation",
            "text": "Realistic images, voices or videos of real people created or edited without their consent; impersonating individuals, organizations or officials to defraud or mislead; forged IDs, news, chat logs or other material made to deceive."
          },
          "copyright": {
            "label": "Copyright / trademark infringement",
            "text": "Copying or substantially reproducing copyrighted works, imitating others' trademarks, logos or brand identity to mislead, sell or promote, and content made to circumvent copyright protection."
          }
        },
        "other": {
          "title": "2. Other Prohibited Uses",
          "p1": "In addition, you must not:",
          "i1": "Use generated content for fraud, gambling, drugs, weapons trading or other illegal activity, or to write malware or carry out cyberattacks",
          "i2": "Mass-produce spam, fake reviews or content intended to manipulate public opinion",
          "i3": "Evade platform or upstream moderation by splitting, encoding or rewording prompts",
          "i4": "Violate the usage policies of upstream model providers (Anthropic, OpenAI, Google, etc.)"
        },
        "moderation": {
          "title": "3. How We Moderate Content",
          "p1": "Generation requests made through the platform are moderated by automated detection combined with human handling:",
          "i1": "Automated detection: chat, code and image generation requests (including reference images uploaded with them) are screened by automated content safety systems covering sexual content, sexual content involving minors, violence and gore, hate, harassment, self-harm and illegal activity, supplemented by a keyword blocklist maintained by the platform.",
          "i2": "Two-stage review: a dedicated content safety service screens first, and a large language model re-checks every hit against the same category rules. Content counts as a violation only when both stages flag it, which reduces false positives.",
          "i3": "Blocking: requests found to violate the policy are blocked with a notice and are not forwarded to the upstream model. The platform can also keep a fingerprint of confirmed violating content and block identical submissions immediately.",
          "i4": "Upstream safeguards: image generation and other requests are also subject to the upstream providers' own safety systems, and requests they refuse produce no output.",
          "i5": "Human handling: categories that automated detection cannot judge reliably, such as deepfakes and infringement, are handled mainly through user reports and manual review. Platform staff review moderation records and investigate reports.",
          "i6": "Moderation records: when a request triggers a moderation rule we record the time, the model, the categories and scores, and an excerpt of at most 240 characters of the input, used to enforce this policy and to review appeals. Records are deleted automatically after the retention period set by the platform (180 days by default). No content is kept for requests that trigger no rule."
        },
        "enforcement": {
          "title": "4. Enforcement",
          "p1": "Depending on the nature and frequency of violations, the platform may:",
          "i1": "Block violating requests and send a warning to the account by site message or email",
          "i2": "Automatically suspend the account once violations within a set period reach the threshold set by the platform",
          "i3": "For severe violations such as child safety content, suspend the account immediately, preserve the relevant records and report to the competent authorities as required by law",
          "i4": "Disable the related API keys or account, or terminate the service; any remaining balance is handled under the Terms of Service"
        },
        "reporting": {
          "title": "5. Reporting Violations",
          "p1": "If you find content generated through the platform that violates this policy (for example, content that misuses your likeness or work), you can report it:",
          "i1": "Registered users: sign in, open Support tickets and submit a ticket in the \"Report prohibited content\" category.",
          "i1Link": "Report now",
          "i2": "Non-users and rights holders: scan the support group QR code at the bottom of this page and in the site footer, or use any other support contact shown there.",
          "p2": "Please include, where possible: screenshots or links to the content, when you found it, the model or request involved, and how to reach you. Rights holders should attach proof of ownership.",
          "p3": "What happens next: we acknowledge the report and investigate it against moderation records and usage logs, normally reaching a decision within 3 business days. Confirmed violations are handled under section 4, and we tell you the outcome through the ticket or the contact you provided. Reports involving child safety are handled first. Reporter information is kept confidential."
        },
        "appeal": {
          "title": "6. Appeals",
          "p1": "If you believe your account was suspended by mistake, sign in as usual on the login page and you will be taken to the appeal page, where you can submit an appeal for manual review. If the suspension was a mistake, we restore the account and notify you by site message."
        },
        "changes": {
          "title": "7. Changes to This Policy",
          "p1": "We may update this policy in response to laws and regulations, upstream provider policies and platform operations. Material changes will be announced by in-app notice or email. Continued use of the service constitutes acceptance of the updated policy."
        }
      },
      "contact": "If you have questions about this policy or need to report prohibited content, contact us through the channels below."
    }
  }
}
