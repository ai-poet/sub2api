// fork 自有的翻译键（上游 locale 中不存在）。
// 单独成文件：同步上游 locale 模块时不会与之冲突，index.ts 里深合并生效。
export default {
  "common": {
    "retry": "Retry"
  },
  "usage": {
    "billingStatus": {
      "pending": "Retrying",
      "failed": "Billing failed",
      "settled": "Settled late",
      "pendingHint": "The billing transaction failed; a background job is retrying with backoff",
      "failedHint": "Retries exhausted or not retryable; needs manual reconciliation",
      "settledHint": "The first billing attempt failed and was settled by the background retry",
      "attempts": "{n} attempts"
    }
  },
  "home": {
    "loginConsole": "Sign in to dashboard",
    "headerTagline": "One-click access to Claude Code / Codex",
    "navModels": "Models",
    "navPricing": "Pricing",
    "navChangelog": "Changelog",
    "hero": {
      "badge": "One-click access to Claude Code / Codex",
      "titleLeadPrimary": "Price-competitive AI relay",
      "titleLeadSecondary": "× Agent desktop client",
      "description": "Deeply unified desktop agent client for Claude Code · Codex · Grok · Pi — sign in and you are routed, with balance, group rates, and uptime at a glance.",
      "primaryNote": "One desktop client for all your agent CLIs",
      "downloadPrimary": "Download now",
      "connectApi": "Use the API",
      "startApi": "Start with the API",
      "badgeDiscount": "vs official API",
      "tags": {
        "coding": "Claude Code",
        "agent": "Codex",
        "tools": "Grok · Pi · More"
      },
      "stats": {
        "setupValue": "1 step",
        "savings": "Auto-configure local tools",
        "routesValue": "2 surfaces",
        "models": "Manage Claude / Codex separately",
        "workspaceValue": "1 place",
        "minCost": "Balance, usage, failure reasons — all here"
      },
      "perToken": "Reference pricing",
      "panel": {
        "title": "A unified entry shaped for daily coding",
        "auto": "Auto Rotating",
        "requestLabel": "request",
        "modelLabel": "models",
        "routeLabel": "route",
        "billLabel": "billing",
        "scenarios": {
          "completions": {
            "title": "One entry for premium-model access, quota control, and automatic switching",
            "subtitle": "Keep the familiar OpenAI chat shape while moving model selection, billing, and route switching into one layer.",
            "route": "Multi-account auto routing",
            "billing": "Pay only for what you use"
          },
          "responses": {
            "title": "Not just chat — tool calls and long-running tasks fit here too",
            "subtitle": "Built for tool-enabled, multi-step workflows without splitting into separate services.",
            "route": "Tool calls + long-running tasks",
            "billing": "Pay only for what you use"
          },
          "messages": {
            "title": "Claude requests can stay inside the same entry",
            "subtitle": "Keep Claude-native habits while reusing unified quotas, routing, and billing.",
            "route": "Claude-native + unified entry",
            "billing": "Pay only for what you use"
          }
        }
      }
    },
    "pricing": {
      "title": "Pay as you go",
      "subtitle": "vs Official Claude Code / Codex API Services",
      "highlight": "Far friendlier pricing than the official API",
      "description": "Run premium models at friendlier prices than the official APIs. One entry covers Claude Code and Codex, balance never expires, you pay only for what you use.",
      "badge": "Premium model access, no longer bound to subscription pricing",
      "barTitle": "Pay-as-you-go billing, balance never expires",
      "points": {
        "metered": "Pay only for what you use — no minimum",
        "routing": "Multi-account auto routing — one route fails, another takes over",
        "visibility": "Usage over the last 5h / 1d / 7d at a glance"
      }
    },
    "download": {
      "badge": "Client download",
      "title": "Download the desktop client",
      "description": "The desktop client auto-configures Claude Code and Codex, safely reuses existing local settings, and brings usage and status into one place.",
      "privacyCode": "Auto-writes local config files without overwriting existing settings",
      "privacyKey": "Desktop, Web, and mobile share one usage and failure feed",
      "cta": "Download",
      "recommended": "Recommended for this device",
      "platforms": {
        "mac": {
          "sub": "Apple Silicon and Intel"
        },
        "windows": {
          "sub": "x64 / ARM64"
        },
        "linux": {
          "sub": ".deb / .rpm / AppImage"
        }
      }
    },
    "comparison": {
      "overline": "A different angle",
    },
    "pricingTable": {
      "overline": "Pricing edge",
      "title": "Friendlier prices than the official APIs for mainstream coding models",
      "description": "Claude Code, Codex, GPT-5.5, and other mainstream coding models — far cheaper than official APIs. One account, one balance, pay only for what you use, balance never expires.",
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
          "title": "Claude Code mainstay",
          "description": "Claude Sonnet 4.6 / Opus 4.7 / Haiku 4.5 — daily coding, refactors, reviews via Claude Code, with per-route spend."
        },
        "codex": {
          "tag": "Codex / GPT family",
          "title": "Codex CLI and GPT-5.5 together",
          "description": "GPT-5.5 alongside GPT-5.4 / GPT-5.3 Codex, local Codex auto-configured — no manual setting changes."
        },
        "compatible": {
          "tag": "OpenAI-compatible · many",
          "title": "Other compatible models, same gateway",
          "description": "Gemini, GLM, Qwen and other compatible models share the same entry, prices reflect the console, failures explained."
        }
      },
      "note": "We do not claim \"lowest price anywhere.\" We focus on total cost: one top-up, pay-as-you-go, per-route usage breakdown, quota and rate-limit status at a glance, clear failure reasons — so AI coding stays sustainable, observable, and diagnosable over the long run. Specific discounts vary by provider and model, and final billing follows the console price."
    },
    "providers": {
      "claudeCode": "Claude Code",
      "gpt": "GPT",
      "codex": "Codex",
      "openaiCompatible": "OpenAI-Compatible"
    },
    "cta": {
      "eyebrow": "READY TO START",
      "stat": "One gateway · Metered billing · Built for daily development"
    }
  },
  "changelog": {
    "overline": "Product / Changelog",
    "headline": "What changed in {siteName}",
    "back": "Back to home",
    "loading": "Loading the changelog…",
    "versions": "{count} releases",
    "latest": "Latest",
    "download": "Download the latest",
    "title": "Changelog",
    "subtitle": "Track what's new and what's changed.",
    "emptyTitle": "No updates yet",
    "emptyDesc": "Check back later for the latest changes and improvements."
  },
  "modelStatus": {
    "title": "Model Status",
    "description": "Check runtime status and recent availability for the groups you can access",
    "featureDisabledTitle": "Model Status Is Not Available",
    "featureDisabledDescription": "The administrator has not enabled this feature yet.",
    "totalGroups": "Monitored Groups",
    "healthyGroups": "Healthy",
    "degradedGroups": "Degraded",
    "downGroups": "Down",
    "lastUpdated": "Last Refresh",
    "autoRefresh": "The list refreshes automatically every 30 seconds",
    "latestProbe": "Latest Probe",
    "latestLatency": "First-token latency",
    "totalLatency": "Total time",
    "availability24": "24h Availability",
    "availability7": "7d Availability",
    "uptimeTimeline": "Last 24 Checks",
    "recentChecksHint": "Each block represents one real probe result",
    "latestResult": "Latest Result",
    "openDetails": "View Details",
    "emptyTitle": "No Visible Groups",
    "emptyDescription": "There are currently no monitored groups available to you.",
    "waiting": "Waiting",
    "waitingForProbe": "No probe has run yet",
    "notAvailable": "N/A",
    "noDescription": "No group description",
    "loadFailed": "Failed to load model status",
    "detailLoadFailed": "Failed to load status details",
    "historyTitle": "History Trend",
    "historyDescription": "Availability and latest state aggregated into time buckets.",
    "eventsTitle": "Recent Events",
    "eventsDescription": "Only stable state changes are recorded as events.",
    "noHistory": "No history is available for the selected period",
    "noEvents": "No recent events",
    "period24h": "24h",
    "period7d": "7d",
    "bucketAvailability": "Bucket Availability",
    "sampleCount": "Samples",
    "avgLatency": "Avg first-token latency",
    "httpCode": "HTTP Code",
    "subStatus": "Sub Status",
    "statuses": {
      "up": "Up",
      "degraded": "Degraded",
      "down": "Down",
      "unknown": "Unknown"
    },
    "eventTypes": {
      "up": "Recovered",
      "down": "Outage",
      "astra_mismatch": "Fingerprint mismatch",
      "astra_recovered": "Fingerprint recovered",
      "modeltrace_mismatch": "Fingerprint mismatch (legacy ModelTrace probe)",
      "modeltrace_recovered": "Fingerprint recovered (legacy ModelTrace probe)",
      "sol_juice_mismatch": "Non-Sol suspected (legacy Juice probe)",
      "sol_juice_recovered": "Sol verified again (legacy Juice probe)"
    },
    "astraCheck": {
      "badge": {
        "pass": "{model} fingerprint matches",
        "mismatch": "{model} fingerprint mismatch (points to {winner})",
        "mismatchNoWinner": "{model} fingerprint mismatch",
        "suspect": "{model} fingerprint suspicious (re-checking)",
        "insufficient": "{model} fingerprint inconclusive",
        "pending": "{model} fingerprint pending"
      },
      "eventMismatch": "{model}: points to {winner}",
      "otherModel": "another model",
      "statuses": {
        "pass": "Fingerprint matches",
        "mismatch": "Fingerprint mismatch",
        "suspect": "Fingerprint suspicious",
        "insufficient": "Fingerprint inconclusive",
        "unknown": "Unknown"
      }
    },
    "benchmarks": {
      "title": "Fingerprint benchmarks",
      "description": "Groups with a fingerprint badge periodically send test prompts upstream to check that the model actually serving them is the one advertised. Each family uses the benchmarks and open-source projects below.",
      "families": {
        "gpt": "GPT family",
        "claude": "Claude family"
      },
      "methods": {
        "meow": "meow benchmark",
        "modeltrace": "ModelTrace number fingerprint",
        "sol_juice": "Juice reading"
      },
      "sources": {
        "gptMeow": "meow v3 benchmark (the official GPT package of meow-llm-detector v4.5.4): a batch of fixed short-answer prompts whose answer distribution is compared with every model in the benchmark; a verdict needs evidence that clearly points at one model.",
        "gptModelTrace": "ModelTrace number fingerprint: 3–6 challenges asking for first-instinct integers from 1 to 355, attributed among the 16 models in the fingerprint bank. GPT-6.1 Sol is not enrolled yet; its number fingerprint is nearly identical to GPT-6 Astra's, so attribution to GPT-6 Astra counts as a match.",
        "gptJuice": "Juice reading: reads the model's internal Juice value at high reasoning; GPT-5.6 Sol should answer 40 (GPT-5.6 Terra 32, Luna 48).",
        "claudeModelTrace": "ModelTrace number fingerprint: the same number challenges, attributed in the same bank (it enrolls 8 Claude models, including Opus 4.6–5.5, Sonnet and Haiku).",
        "claudeMeow": "meow v3 benchmark (the official Claude package of meow-llm-detector v4.5.4): answer distributions of fixed short-answer prompts."
      },
      "repo": "Source",
      "site": "Web version",
      "license": "License",
      "selfImplemented": "Implemented by this site",
      "disclaimer": "Checks only decide among the models a benchmark enrolls: a mismatch means the answers clearly resemble another enrolled model. It is a signal, not proof of substitution on its own, and a model outside the benchmark may be attributed to its closest candidate."
    }
  },
  "nav": {
    "integrationGuide": "Integration Guide",
    "referralSettings": "Referral",
    "dataManagement": "Data Management",
    "paymentManagement": "Payment Management",
    "referral": "Referral",
    "modelMirror": "Claude Relay Inspector",
    "modelCatalog": "Model Catalog",
    "modelStatus": "Model Status",
    "sora": "Sora Studio",
    "communityGroup": "Join Community",
    "communityGroupTooltip": "Scan to join our community group",
    "communityGroupScanHint": "Scan the QR code with your phone",
    "communityGroupJoin": "Join Now",
    "approvals": "Approvals",
    "tickets": "Tickets"
  },
  "auth": {
    "referralCodeLabel": "Referral Code",
    "referralCodePlaceholder": "Enter referral code (optional)",
    "github": {
      "signIn": "Continue with GitHub",
      "orContinue": "or continue with email",
      "callbackTitle": "Signing you in",
      "callbackProcessing": "Completing login, please wait...",
      "callbackHint": "If you are not redirected automatically, go back to the login page and try again.",
      "callbackMissingToken": "Missing login token, please try again.",
      "backToLogin": "Back to Login",
      "invitationRequired": "This GitHub account is not yet registered. The site requires an invitation code — please enter one to complete registration.",
      "invalidPendingToken": "The registration token has expired. Please sign in with GitHub again.",
      "completeRegistration": "Complete Registration",
      "completing": "Completing registration…",
      "completeRegistrationFailed": "Registration failed. Please check your invitation code and try again."
    },
    "desktopBridge": {
      "pageTitle": "Desktop Sign-in",
      "title": "Connecting the desktop app",
      "preparing": "Preparing your session for the desktop app…",
      "preparingRoutes": "Preparing Claude Code and Codex routes…",
      "creatingSession": "Creating a session for the app…",
      "creatingCode": "Creating a sign-in code…",
      "opening": "Opening the app…",
      "manualHint": "If the app did not open automatically, continue manually:",
      "openApp": "Open the app",
      "failed": "Unable to complete the desktop sign-in.",
      "failedStatus": "Unable to continue automatically.",
      "code": {
        "label": "Sign-in code",
        "copy": "Copy",
        "copied": "Copied",
        "hint": "If the app did not finish signing in automatically, copy this code and paste it into the app's sign-in window (you can also paste the whole link from the address bar).",
        "expiresIn": "Valid for {time}, single use",
        "warning": "Only paste this code into the app you just started signing in from. Never send it to anyone — whoever redeems it gets into your account.",
        "expired": "This code has expired — start signing in again from the app.",
        "returnToApp": "Return to the app"
      }
    }
  },
  "integrationGuide": {
    "title": "Integration Guide",
    "description": "Browse platform-specific client setup and copy ready-to-use configs",
    "caption": "Multi-platform setup",
    "intro": "This page organizes the common client setup flows by platform. Each platform can pick one of your existing API keys and render copy-ready configuration snippets immediately.",
    "bindingNote": "Each API key is bound to one group platform and cannot be reused across every platform. Platforms without an available key stay visible in example mode until you create a matching key.",
    "liveBadge": "Live Config",
    "exampleBadge": "Example Mode",
    "keyLabel": "Current API Key",
    "selectKey": "Select an API key for this platform",
    "keyHelp": "A real key is already injected into the snippets and can be copied directly.",
    "noKeysForPlatform": "No API key is currently available for this platform.",
    "exampleOption": "Show example config",
    "exampleDescription": "The snippets below are placeholders. Once you create a key for this platform, this panel will switch to live config automatically.",
    "failedToLoadKeys": "Failed to load API keys for the integration guide",
    "failedToLoadSettings": "Failed to load public settings for the integration guide",
    "platforms": {
      "anthropic": "Anthropic",
      "openai": "OpenAI"
    }
  },
  "redeem": {
    "referralReward": "Referral Reward"
  },
  "referral": {
    "title": "Referral",
    "description": "Invite friends to register and recharge, both parties will receive rewards",
    "yourLink": "Your Referral Link",
    "copyLink": "Copy Link",
    "copied": "Copied",
    "code": "Referral Code",
    "totalInvited": "Total Invited",
    "rewarded": "Rewarded",
    "pending": "Pending",
    "totalEarned": "Total Earned",
    "history": "Referral History",
    "referee": "Referred User",
    "status": "Status",
    "reward": "Reward",
    "time": "Time",
    "noHistory": "No referral records yet",
    "statusRewarded": "Rewarded",
    "statusPending": "Pending Recharge",
    "days": " days",
    "loadFailed": "Failed to load referral info",
    "copyFailed": "Copy failed, please copy manually",
    "rulesTitle": "Referral Reward Rules",
    "rule1": "Share your referral link with friends. A referral relationship is established when they register through your link",
    "rule2": "Both parties receive rewards after the referred user makes their first recharge",
    "rule3": "Rewards are automatically credited to account balance or subscription duration",
    "ruleReferrerReward": "Referrer reward",
    "ruleRefereeReward": "Referee reward"
  },
  "admin": {
    "users": {
      "typeReferralReward": "Balance (Referral Reward)",
      "siteMessage": {
        "menuItem": "Send site message",
        "title": "Send a site message to {email}",
        "recipient": "Recipient",
        "titleLabel": "Title",
        "titlePlaceholder": "e.g. Account usage reminder",
        "contentLabel": "Content (Markdown supported)",
        "contentPlaceholder": "What you want to tell the user. It pops up the next time they open the site.",
        "edit": "Edit",
        "preview": "Preview",
        "previewEmpty": "Nothing to preview",
        "send": "Send",
        "sent": "Site message sent",
        "operatorHint": "Messages sent by an operator are delivered after an admin approves them.",
        "disabledHint": "This account is disabled. The user can read site messages on the appeal page.",
        "titleRequired": "Title is required",
        "contentRequired": "Content is required",
        "titleTooLong": "Title must be at most {max} characters",
        "contentTooLong": "Content must be at most {max} characters",
        "history": "Message history",
        "historyEmpty": "This user has not received any site messages",
        "read": "Read",
        "unread": "Unread",
        "readAt": "Read {time}",
        "senderSystem": "System",
        "viaApproval": "Approval #{id}",
        "loadFailed": "Failed to load message history",
        "showContent": "Show content",
        "hideContent": "Hide content"
      }
    },
    "riskControl": {
      "siteMessageOnHit": "Send site message on hit",
      "siteMessageOnHitHint": "Send the user a site message (shown as a popup) when a request hits a risk rule, the account is auto-disabled, or the cyber-security policy blocks a request. Works without email settings. When off, content audit sends no site messages at all, including disable notices.",
      "viewSettings": "View settings",
      "readonlyNotice": "Operator read-only: you can view the content audit settings, runtime status and audit records. Changing settings, testing keys, unbanning users and clearing flagged hashes are left to the admin. Audit engine keys are not shown to operators."
    },
    "groups": {
      "columns": {
        "runtimeStatus": "Runtime Status"
      },
      "runtimeStatus": {
        "action": "Runtime Status",
        "title": "{name} Runtime Status",
        "titlePlain": "Runtime Status",
        "enabled": "Enable group runtime monitoring",
        "enabledHint": "When disabled, this group is not probed in the background and will not be shown to regular users.",
        "probeModel": "Probe Model",
        "probeModelPlaceholder": "e.g. gpt-4.1-mini",
        "probePrompt": "Probe Prompt",
        "probePromptPlaceholder": "Enter the prompt used for the minimal probe request",
        "validationMode": "Validation Mode",
        "validationModes": {
          "nonEmpty": "Non-empty response",
          "keywordsAny": "Match any keyword",
          "keywordsAll": "Match all keywords"
        },
        "expectedKeywords": "Expected Keywords",
        "expectedKeywordsPlaceholder": "Separate keywords with commas or new lines",
        "expectedKeywordsHint": "Only used for keyword validation modes. Duplicates are removed automatically.",
        "intervalSeconds": "Probe Interval (seconds)",
        "timeoutSeconds": "Timeout Threshold (seconds)",
        "slowLatencyMs": "Slow threshold (ms, first token)",
        "latestResult": "Latest Result Preview",
        "latestResultHint": "Shows the latest stable state and raw probe summary.",
        "latestResultEmpty": "No probe result yet. Save the config or run an immediate probe to populate this area.",
        "currentStatus": "Current Status",
        "checkedAt": "Checked At",
        "latency": "First-token latency",
        "totalLatency": "Total time",
        "httpCode": "HTTP",
        "subStatus": "Sub Status",
        "responseExcerpt": "Response Excerpt",
        "errorDetail": "Error Detail",
        "save": "Save Config",
        "saved": "Runtime status config saved",
        "failedToLoad": "Failed to load runtime status config",
        "failedToSave": "Failed to save runtime status config",
        "probeNow": "Probe Now",
        "probing": "Probing...",
        "probeSucceeded": "Immediate probe completed",
        "probeFailed": "Immediate probe failed",
        "disabled": "Disabled",
        "waiting": "Waiting",
        "notConfigured": "Not Configured",
        "footerHint": "\"Probe now\" saves the current form first and then runs a probe immediately.",
        "notifyEnabled": "Push on down / recovery",
        "notifyEnabledHint": "Only effective when Server酱³ push is enabled in site settings; turn off to silence this group.",
        "astraCheck": {
          "title": "meow fingerprint check",
          "hint": "Sends a batch of fixed short-answer prompts to one account of this group that matches its platform and scores the answer distribution against the bundled meow v3 benchmark: every candidate accumulates evidence per prompt, and only a candidate whose evidence is uniquely highest and clears its own strong-direction line counts as a strong match. Several expected models can be checked at once; each is judged and pushed on its own. Two consecutive strong matches on another model (the first triggers an immediate re-run on the same account) flip the verdict; availability is not affected. GPT-5.6 Sol uses the Juice reading instead (one request at reasoning=high, Sol should answer 40) and ignores the tier; Claude Opus 5.5, Opus 5 and GPT-6.1 Sol use the ModelTrace number fingerprint (3–6 long number challenges, attributed against the prompts and bank of the official web version), which also ignores the tier; GPT-6.1 Sol is not in the bank, so attribution to GPT-6 Astra, whose number fingerprint is nearly identical, counts as a match. The default medium tier (recommended by meow) sends 64–96 requests per model; the low tier halves that but lets models with spread-out answers such as Sol drift more easily.",
          "benchmark": "Benchmark",
          "methods": {
            "meow": "meow benchmark",
            "sol_juice": "Juice reading",
            "modeltrace": "ModelTrace"
          },
          "juiceHint": "Reads Juice at high reasoning · Sol should answer 40 (Terra 32 / Luna 48)",
          "modeltraceHint": "ModelTrace number fingerprint · 3 valid outputs (up to 6 challenges) · closed-set attribution",
          "modeltraceProxyHint": "{model} is not in the bank and is judged by {proxy}, whose number fingerprint is nearly identical: attribution to {proxy} counts as a match",
          "modeltraceProbabilities": "Attribution probabilities (top 5 plus the expected model; match line 50%, strong line 80%)",
          "models": "Models to check",
          "modelsHint": "Tick the expected models to verify; the field on the right is the model name sent upstream (empty = default).",
          "requestModel": "Request model",
          "noModelSelected": "Select at least one model when the fingerprint check is enabled",
          "tier": "Tier",
          "tiers": {
            "low": "Low",
            "medium": "Medium",
            "high": "High"
          },
          "intervalSeconds": "Check interval (seconds, min 900)",
          "latestResult": "Latest check per model",
          "latestResultEmpty": "No models to check yet. Tick models and save, then wait for the scheduler or click \"Check all\".",
          "notChecked": "Not checked yet. Save and wait for the scheduler, or click \"Check this model\".",
          "samples": "Valid samples / planned",
          "checkedAt": "Checked at",
          "tokens": "Tokens (input / output)",
          "lastCost": "Last run cost",
          "monthlyEstimate": "Monthly at current interval",
          "monthlyTotal": "All models, monthly at current interval",
          "matches": "Candidate match scores",
          "threshold": "strong line",
          "detail": "Details",
          "otherModel": "Other model",
          "disclaimer": "The verdict compares candidates inside the benchmark package and is indicative only: models outside the package fall into \"other\" or the closest candidate, and the match score is the average advantage per answer over the strongest rival, not an identity probability.",
          "probeAll": "Check all",
          "probeModel": "Check this model",
          "running": "Checking...",
          "probeStarted": "meow fingerprint check started in the background",
          "probeSucceeded": "meow fingerprint check finished",
          "probeFailed": "meow fingerprint check failed",
          "progress": {
            "title": "Checking {model} ({index}/{count}) · round {round}",
            "phases": {
              "selecting_account": "Selecting an account and sending the first request",
              "running": "Sending requests",
              "scoring": "Scoring"
            },
            "account": "Account",
            "valid": "valid",
            "invalid": "invalid",
            "failed": "failed",
            "requests": "requests",
            "inFlight": "in flight"
          },
          "sampleTable": {
            "title": "Per-request samples of the latest run ({count})",
            "runMeta": "completed {completed}/{planned}, took {latency}",
            "seq": "#",
            "cell": "Prompt",
            "attempt": "Attempt",
            "answer": "Answer",
            "category": "Normalized",
            "latency": "Latency"
          },
          "statuses": {
            "pass": "Fingerprint matches",
            "mismatch": "Fingerprint mismatch (points to {winner})",
            "mismatchNoWinner": "Fingerprint mismatch",
            "suspect": "Suspicious (points to {winner}, re-checking)",
            "suspectNoWinner": "Suspicious (re-checking)",
            "insufficient": "Inconclusive",
            "unknown": "Pending"
          },
          "reasons": {
            "samples_incomplete": "Not enough valid samples",
            "no_valid_samples": "No valid samples",
            "no_strong_direction": "No candidate reached a strong direction",
            "uncalibrated": "This tier is not calibrated",
            "target_not_allowed": "Expected model does not apply to this platform",
            "target_not_in_benchmark": "The benchmark does not contain this model",
            "benchmark_invalid": "Benchmark unavailable",
            "no_account": "No available account",
            "juice_inconclusive": "Juice reading inconclusive",
            "unknown_expected_model": "Expected model is not in the ModelTrace bank",
            "no_valid_outputs": "No valid outputs",
            "insufficient_outputs": "Fewer than 2 valid outputs",
            "ambiguous": "Attribution is ambiguous",
            "bank_invalid": "ModelTrace bank unavailable",
            "scoring_failed": "Scoring failed",
            "empty_result": "Empty result"
          }
        }
      },
      "openaiMessages": {
        "defaultModel": "Default mapped model",
        "defaultModelPlaceholder": "e.g., gpt-4.1",
        "defaultModelHint": "When account has no model mapping configured, all request models will be mapped to this model"
      }
    },
    "accounts": {
      "sendingGeminiImageRequest": "Sending Gemini image generation test request...",
      "geminiImagePromptLabel": "Image prompt",
      "geminiImagePromptPlaceholder": "Example: Generate an orange cat astronaut sticker in pixel-art style on a solid background.",
      "geminiImagePromptDefault": "Generate a cute orange cat astronaut sticker on a clean pastel background.",
      "geminiImageTestHint": "When a Gemini image model is selected, this test sends a real image-generation request and previews the returned image below.",
      "geminiImageTestMode": "Mode: Gemini image generation test",
      "geminiImagePreview": "Generated images:",
      "geminiImageReceived": "Received test image #{count}"
    },
    "redeem": {
      "types": {
        "referral_reward": "Referral Reward"
      }
    },
    "ops": {
      "alertRules": {
        "metrics": {
          "billingFailureCount": "Billing failures",
          "billingUnsettledCount": "Unsettled billings"
        },
        "metricDescriptions": {
          "billingFailureCount": "Requests whose billing transaction failed and entered the retry queue within the window; any value means the usage page needs a look.",
          "billingUnsettledCount": "Requests currently pending a billing retry plus those whose retries were abandoned (not limited to the window)."
        }
      },
      "errorDetail": {
        "pinnedToOriginalAccountId": "Pinned to original account_id",
        "missingUpstreamRequestBody": "Missing upstream request body",
        "failedToLoadRetryHistory": "Failed to load retry history",
        "unsupportedRetryMode": "Unsupported retry mode",
        "classificationKeys": {
          "retryable": "Retryable",
          "resolvedRetryId": "Resolved Retry",
          "retryCount": "Retry Count"
        },
        "retryMeta": {
          "used": "Used",
          "success": "Success",
          "pinned": "Pinned"
        },
        "notRetryable": "Not recommended to retry",
        "retry": "Retry",
        "retryClient": "Retry (Client)",
        "retryUpstream": "Retry (Upstream pinned)",
        "pinnedAccountId": "Pinned account_id",
        "retryNotes": "Retry Notes",
        "requestBody": "Request Body",
        "confirmRetry": "Confirm Retry",
        "retrySuccess": "Retry succeeded",
        "retryFailed": "Retry failed",
        "retryHint": "Retry will resend the request with the same parameters",
        "retryClientHint": "Use client retry (no account pinning)",
        "retryUpstreamHint": "Use upstream pinned retry (pin to the error account)",
        "pinnedAccountIdHint": "(auto from error log)",
        "retryNote1": "Retry will use the same request body and parameters",
        "retryNote2": "If the original request failed due to account issues, pinned retry may still fail",
        "retryNote3": "Client retry will reselect an account",
        "retryNote4": "You can force retry for non-retryable errors, but it is not recommended",
        "confirmRetryMessage": "Confirm retry this request?",
        "confirmRetryHint": "Will resend with the same request parameters",
        "forceRetry": "I understand and want to force retry",
        "forceRetryHint": "This error usually cannot be fixed by retry; check to proceed",
        "forceRetryNeedAck": "Please check to force retry",
        "viewRetries": "Retry history",
        "retryHistory": "Retry History",
        "tabRetries": "Retries",
        "retrySummary": "Retry Summary",
        "responseHintSucceeded": "Showing succeeded retry response_preview (#{id})",
        "responseHintFallback": "No succeeded retry found; showing stored error_body",
        "suggestUpstreamResolved": "✓ Upstream error resolved by retry; no action needed"
      },
      "settings": {
        "ignoreInvalidApiKeyErrors": "Ignore invalid API key errors",
        "ignoreInvalidApiKeyErrorsHint": "When enabled, invalid or missing API key errors (INVALID_API_KEY, API_KEY_REQUIRED) will not be written to the error log."
      }
    },
    "audit": {
      "filters": {
        "personalToken": "Personal token",
        "appealToken": "Appeal session"
      }
    },
    "referral": {
      "title": "Referral Settings",
      "description": "Configure referral reward system parameters",
      "enabled": "Enable Referral System",
      "enabledDesc": "When enabled, users can invite friends to register via referral links",
      "maxPerUser": "Max Referrals Per User",
      "maxPerUserHint": "0 means unlimited",
      "referrerRewards": "Referrer Rewards",
      "refereeRewards": "Referee Rewards",
      "balanceReward": "Balance Reward",
      "groupId": "Subscription Group",
      "groupIdHint": "Select \"None\" for no subscription reward",
      "noGroup": "None (no subscription reward)",
      "subscriptionDays": "Subscription Days",
      "saved": "Referral settings saved",
      "saveFailed": "Failed to save referral settings",
      "loadFailed": "Failed to load referral settings"
    },
    "settings": {
      "tabs": {
        "client": "Client",
        "data": "Sora Storage"
      },
      "github": {
        "title": "GitHub Login",
        "description": "Configure GitHub OAuth for Sub2API end-user login",
        "enable": "Enable GitHub Login",
        "enableHint": "Show GitHub login on the login/register pages",
        "clientId": "Client ID",
        "clientIdPlaceholder": "Iv1.1234567890abcdef",
        "clientIdHint": "Get this from GitHub Developer Settings",
        "clientSecret": "Client Secret",
        "clientSecretPlaceholder": "********",
        "clientSecretHint": "Used by backend to exchange tokens (keep it secret)",
        "clientSecretConfiguredPlaceholder": "********",
        "clientSecretConfiguredHint": "Secret configured. Leave empty to keep the current value.",
        "redirectUrl": "Redirect URL",
        "redirectUrlPlaceholder": "https://your-domain.com/api/v1/auth/oauth/github/callback",
        "redirectUrlHint": "Must match the callback URL configured in your GitHub OAuth App (must be an absolute http(s) URL)",
        "quickSetCopy": "Generate & Copy (current site)",
        "redirectUrlSetAndCopied": "Redirect URL generated and copied to clipboard"
      },
      "site": {
        "groupStatusEnabled": "Enable Model Status",
        "groupStatusEnabledDescription": "When enabled, regular users can see the \"Model Status\" menu and use the corresponding runtime status APIs.",
        "groupStatusNotify": {
          "title": "Server酱³ Push",
          "description": "Push a notification when a group's stable status turns red or recovers from red. Enable this switch and fill in the UID and SendKey; each group can opt out in its runtime status config.",
          "enabled": "Enable Server酱³ push",
          "uid": "UID",
          "uidPlaceholder": "e.g. 12345",
          "uidHint": "Push host is {uid}.push.ft07.com; only letters, digits, - and _ are allowed.",
          "sendkey": "SendKey",
          "sendkeyPlaceholder": "sctp...",
          "sendkeyConfiguredPlaceholder": "Configured. Leave empty to keep the current value.",
          "sendkeyHint": "Stored on the backend only and never echoed back.",
          "test": "Send test push",
          "testing": "Sending...",
          "testHint": "Uses the UID / SendKey above; an empty SendKey falls back to the saved one.",
          "testSucceeded": "Test push sent",
          "testFailed": "Test push failed"
        },
        "approvalNotify": {
          "enabled": "Approval request push",
          "enabledHint": "Push a message with a link to the approvals page whenever an operator submits a user / subscription management change (uses the UID / SendKey above)."
        },
        "approvalLimits": {
          "title": "Operator approval limits",
          "description": "Operator write requests wait in the approval queue; tune the queue and batch-approve limits here. Changes apply immediately after saving.",
          "pendingPerUser": "Pending requests per operator",
          "pendingPerUserHint": "How many pending requests one operator may have at the same time; further requests are refused (1–1000, default 20).",
          "batch": "Batch approve size",
          "batchHint": "How many requests \"Approve selected / Approve all pending\" execute per batch; larger sets are split automatically (1–500, default 50)."
        },
        "ticketNotify": {
          "enabled": "Ticket push",
          "enabledHint": "Push a message with a link to the ticket whenever a user opens a ticket or adds a reply (uses the UID / SendKey above); staff replies are not pushed."
        },
        "billingFailureNotify": {
          "enabled": "Billing failure push",
          "enabledHint": "Push a summary when a request's billing fails and is queued for retry, or when a retry is abandoned (uses the UID / SendKey above); at most one push per 10 minutes per instance."
        },
        "personalToken": {
          "title": "Operator personal tokens",
          "description": "When enabled, operators can generate a personal token on their profile page so scripts can call the API with an Authorization: Bearer header. On admin endpoints a token has exactly the same access as the operator's browser session (same allowlist; user and subscription writes still go through approval). User endpoints act on the operator's own account, except account security actions (password, 2FA, passkeys, login bindings, the token itself). A token cannot pass two-factor step-up and cannot approve requests. Turning this off disables every token immediately without deleting them; turning it back on restores them.",
          "listTitle": "Issued tokens",
          "refresh": "Refresh",
          "empty": "No operator has generated a token yet",
          "loadFailed": "Failed to load tokens",
          "createdAt": "Created {date}",
          "expiresAt": "Expires {date}",
          "neverExpires": "Never expires",
          "lastUsed": "Last used {date} ({ip})",
          "neverUsed": "Never used",
          "revoke": "Revoke",
          "revokeTitle": "Revoke personal token",
          "revokeConfirm": "Scripts using the token of {email} will stop working immediately and a new token must be generated. Revoke it?",
          "revoked": "Token revoked",
          "revokeFailed": "Failed to revoke token",
          "states": {
            "active": "Active",
            "expired": "Expired",
            "revoked": "Invalid (password or email changed)",
            "not_eligible": "Invalid (no longer an operator)",
            "user_inactive": "Invalid (account disabled)",
            "user_missing": "Invalid (account missing)"
          }
        },
        "contentTranslation": {
          "title": "Content auto-translation",
          "description": "Uses one of your own API keys and a cheap model to translate admin-written copy (group names and descriptions, announcements, site subtitle, contact info, channel descriptions, custom menus and endpoints, login agreements, payment plans) into the selected languages, cached permanently. Only the text users see is replaced; API responses and routing stay unchanged, and the admin console always shows the original. The switch saves as soon as you flip it; the other fields save with “Save translation settings” or the Save button at the bottom of the page.",
          "enabled": "Enable auto-translation",
          "apiKey": "API key to call with",
          "apiKeyPlaceholder": "Choose one of your own API keys",
          "apiKeyHint": "Only the key id is stored. The key must belong to an admin and be active, and its group must serve the model below.",
          "apiKeyInactive": "inactive",
          "apiKeyMissing": "unavailable",
          "apiKeysLoadFailed": "Failed to load API keys",
          "model": "Model",
          "modelPlaceholder": "e.g. gpt-5.4-mini",
          "modelHint": "A cheap, fast model is enough; unchanged copy never calls the model again.",
          "baseUrl": "Gateway URL (optional)",
          "baseUrlPlaceholder": "Blank = this gateway via loopback",
          "baseUrlHint": "Only {path} is requested. Blank uses loopback http://127.0.0.1:<port>; a public URL also works.",
          "languages": "Target languages",
          "languageNames": {
            "zh": "中文",
            "en": "English",
            "ja": "日本語"
          },
          "save": "Save translation settings",
          "saving": "Saving...",
          "saved": "Translation settings saved",
          "saveFailed": "Failed to save translation settings",
          "loadFailed": "Failed to load translation settings",
          "apiKeyRequired": "Choose an API key before enabling",
          "modelRequired": "Enter a model before enabling",
          "languagesRequired": "Select at least one target language",
          "test": "Test",
          "testing": "Testing...",
          "testFailed": "Test translation failed",
          "testResult": "Sample translation ({ms} ms)",
          "sync": "Sync now",
          "syncing": "Syncing...",
          "syncStarted": "Scan started; the status refreshes automatically",
          "syncFailed": "Failed to start a sync",
          "syncDisabled": "Auto-translation is still disabled on the server, so this scan only collected the sources and translated nothing. Turn on the switch above (it saves at once), then sync again.",
          "unsaved": "Unsaved changes: click “Save translation settings” on the right or Save at the bottom of the page",
          "manage": "Manage translations",
          "refreshStatus": "Refresh status",
          "status": {
            "enabled": "Enabled",
            "disabled": "Disabled",
            "summary": "{sources} sources · {translated} translations · {pending} pending",
            "running": "Translating…",
            "lastRun": "Last run: {time}",
            "neverRun": "Not run yet",
            "lastError": "Last error ({time}): {error}",
            "loadFailed": "Failed to load the status"
          },
          "dialog": {
            "title": "Manage translations",
            "allLanguages": "All languages",
            "searchPlaceholder": "Search source or translation",
            "refresh": "Refresh",
            "columns": {
              "source": "Source",
              "translation": "Translation",
              "lang": "Language",
              "model": "Model",
              "flags": "Flags",
              "actions": "Actions"
            },
            "manual": "Manual",
            "inUse": "In use",
            "notInUse": "Not in use",
            "notInUseHint": "The source text is not currently collected. The translation is kept and reused if the text comes back.",
            "edit": "Edit",
            "save": "Save",
            "cancel": "Cancel",
            "delete": "Delete",
            "loadFailed": "Failed to load translations",
            "updated": "Translation updated and marked as manual",
            "updateFailed": "Failed to update the translation",
            "emptyTranslation": "The translation cannot be empty",
            "deleteTitle": "Delete translation",
            "deleteConfirm": "This text will be translated again in the next run. Delete it?",
            "deleted": "Translation deleted",
            "deleteFailed": "Failed to delete the translation",
            "clearMachine": "Clear machine translations",
            "clearTitle": "Clear machine translations",
            "clearConfirm": "All machine translations will be deleted (manual ones are kept) and re-translated with the current model. Clear them?",
            "cleared": "Cleared {count} machine translations",
            "clearFailed": "Failed to clear machine translations"
          }
        },
        "communityQRCodePlaceholder": "Paste the QR image base64 or URL",
        "communityQRCode": "Community Group QR Code",
        "uploadQRCode": "Upload QR Code",
        "qrCodeHint": "Upload a QR code image for the community group. Max 500KB. Once uploaded, a community entry will appear in the top navigation bar.",
        "communityGroupURL": "Community Group URL",
        "communityGroupURLPlaceholder": "https://t.me/example",
        "communityGroupURLHint": "Optional. If provided, a join link will be shown below the QR code. Must be an absolute http(s) URL."
      },
      "purchase": {
        "openMode": "Open Mode",
        "openModeIframe": "Embedded (iframe)",
        "openModeNewWindow": "New Window",
        "openModeHint": "Choose how to open the recharge/orders page"
      },
      "clientDownloads": {
        "title": "Client Downloads",
        "description": "Set public Windows and macOS desktop client download links. The default home page shows a download entry when at least one link is configured. When both are empty, the home page hides all client-related content and the primary button becomes \"Start with the API\".",
        "windowsUrl": "Windows Download URL",
        "windowsUrlPlaceholder": "https://downloads.example.com/sub2api-windows.exe",
        "windowsUrlHint": "Leave empty to hide the Windows download button.",
        "macosUrl": "macOS Install Command",
        "macosUrlPlaceholder": "curl -fsSL https://example.com/install.sh | bash",
        "macosUrlHint": "Enter a terminal install command. Users click to copy it and run it in their terminal. Leave empty to hide the macOS install entry.",
        "publicHint": "Use a public http(s) link from object storage, a CDN, or a release platform. Custom home page content still fully controls the home page when configured.",
        "changelogRepo": "Changelog source (GitHub repository)",
        "changelogRepoHint": "The /changelog page syncs this repository's GitHub Releases automatically (drafts and pre-releases are skipped) and refreshes every 15 minutes. Enter owner/repo or the repository URL; leave empty to use {repo}."
      }
    }
  },
  "modelCatalog": {
    "title": "Model Catalog",
    "description": "Compare the official reference price with your actual group-level charge for models you can truly access.",
    "caption": "Group x Model Price Cards",
    "intro": "Each card represents one accessible \"group + model\" combination. The page focuses on a direct official-vs-displayed-price comparison and defaults to the lowest displayed price first.",
    "lastUpdated": "Last Updated",
    "neverUpdated": "Not loaded yet",
    "paymentNoticeTitle": "Actual payment price unavailable",
    "paymentNoticeDescription": "Payment conversion config could not be loaded. Showing only official price and USD balance charge.",
    "filters": {
      "search": "Search",
      "searchPlaceholder": "Search model, group, or platform",
      "platform": "Platform",
      "allPlatforms": "All platforms",
      "billingMode": "Billing mode",
      "allBillingModes": "All billing modes",
      "sortBy": "Sort by"
    },
    "sorting": {
      "effectivePriceAsc": "Lowest displayed price",
      "modelAsc": "Model name"
    },
    "groupTabs": {
      "title": "Browse by group",
      "description": "Use group tabs as the primary switcher. Search and other filters stay available as secondary tools.",
      "allGroups": "All groups",
      "currentGroup": "Current group: {group}"
    },
    "filterResult": "Showing {visible} / {total} cards",
    "priceBasis": "Official reference vs actual payment price (falls back to balance price)",
    "cnyRateReady": "Payment conversion ready at ¥{rate} per $1 balance",
    "loadFailedTitle": "Failed to load model catalog",
    "loadFailedDescription": "The catalog is temporarily unavailable. Refresh and try again.",
    "emptyTitle": "No matching cards",
    "emptyDescription": "Try broader filters or switch to a different group.",
    "billingMode": {
      "token": "Token",
      "perRequest": "Per request",
      "image": "Per image",
      "video": "Per second"
    },
    "rateSource": {
      "groupDefault": "Group default multiplier",
      "userOverride": "User override multiplier"
    },
    "referenceSource": {
      "litellm": "LiteLLM reference",
      "fallback": "Fallback reference",
      "none": "No reference"
    },
    "groupRateLabel": "Rate {rate}",
    "peerGroupsLabel": "{count} groups carry this model",
    "primaryPrice": "Primary price",
    "priceLabels": {
      "input": "Input",
      "output": "Output",
      "cacheWrite": "Cache write",
      "cacheRead": "Cache read",
      "perRequest": "Per request",
      "perImage": "Per image",
      "perSecond": "Per second"
    },
    "units": {
      "perMillionTokens": "Per 1M tokens",
      "perRequest": "Per request",
      "perImage": "Per image",
      "perSecond": "Per second of video"
    },
    "priceColumns": {
      "official": "Official",
      "balance": "Balance",
      "cash": "Actual paid"
    },
    "capabilities": {
      "promptCaching": "Prompt caching",
      "longContext": "Long context threshold {threshold}",
      "tieredPricing": "{count} pricing tiers",
      "resolutionTiers": "{count} resolution tiers",
      "userRateOverride": "User override"
    },
    "defaultTierHint": "Billed at this tier when no size is given",
    "expandDetails": "Expand tiers and peer groups",
    "collapseDetails": "Collapse details",
    "intervalSectionTitle": "Channel tier pricing",
    "intervalDefaultLabel": "Default tier",
    "otherGroupsTitle": "Other accessible groups for this model",
    "peerDisplayedPrice": "Displayed price for this group"
  },
  "announcements": {
    "newAnnouncement": "New Announcement"
  },
  "operator": {
    "personalToken": {
      "title": "Personal token",
      "description": "Lets scripts call the API for automation (both admin and user endpoints): send Authorization: Bearer followed by the token. It has the same access as your browser session.",
      "generate": "Generate token",
      "regenerate": "Regenerate",
      "revoke": "Revoke",
      "featureDisabled": "The administrator has not enabled personal tokens yet. Existing tokens also stop working while the feature is off.",
      "empty": "No personal token yet",
      "showOnceWarning": "Copy the token now and store it safely. It is shown only once and cannot be viewed again after you leave or reload this page.",
      "copy": "Copy",
      "copied": "Token copied",
      "usageHint": "Example:",
      "done": "I have saved it",
      "expired": "Expired",
      "createdAt": "Created {date}",
      "expiresAt": "Expires {date}",
      "neverExpires": "Never expires",
      "lastUsed": "Last used {date} ({ip})",
      "neverUsed": "Never used",
      "generateTitle": "Generate personal token",
      "regenerateTitle": "Regenerate personal token",
      "regenerateWarning": "The old token stops working immediately; scripts using it must be updated.",
      "expiry": "Expiration",
      "days": "{days} days",
      "confirmGenerate": "Generate",
      "generated": "Token generated",
      "generateFailed": "Failed to generate token",
      "loadFailed": "Failed to load personal token",
      "revokeTitle": "Revoke personal token",
      "revokeConfirm": "Scripts using this token will stop working immediately. Revoke it?",
      "revoked": "Token revoked",
      "revokeFailed": "Failed to revoke token",
      "notes": {
        "scope": "It works on the admin and user endpoints you can use in the browser (e.g. /auth/me, group status, API keys); anything beyond your access returns 403.",
        "approval": "User and subscription management writes still go to the approval queue (202) and run only after the administrator approves them.",
        "invalidation": "Changing your password or email, revocation by the administrator, a role change, or the administrator turning the feature off invalidates the token immediately.",
        "noSensitive": "So that a leaked token cannot take over the account, it cannot perform account security actions: changing the password, 2FA, passkeys, login bindings, generating or revoking tokens, or issuing login sessions. It also cannot perform actions that require two-factor verification."
      }
    },
    "readOnlyNotice": "Read-only mode: operators can view ops monitoring and usage logs but cannot change settings, handle alerts or clean up data.",
    "roleHint": "Operator: troubleshooting role with access to ops monitoring and usage logs; user and subscription management changes only execute after admin approval, and upstream accounts or groups cannot be managed.",
    "singleAdminHint": "Only one admin account is allowed; use the operator or user role for everyone else.",
    "approval": {
      "title": "Approvals",
      "description": "User / subscription management changes submitted by operators, executed once the admin approves them with one click.",
      "queuedToast": "Submitted for admin approval: {target}",
      "operatorHint": "Only your own requests are listed here; approved requests run under the admin's identity.",
      "approving": "Executing...",
      "approveSuccess": "Approved and executed: {target}",
      "approveFailed": "Approved but execution failed: {error}",
      "rejected": "Request rejected",
      "cancelled": "Request withdrawn",
      "rejectTitle": "Reject request",
      "rejectReasonPlaceholder": "Reason (optional, visible to the requester)",
      "cancelTitle": "Withdraw request",
      "cancelConfirm": "Withdraw this pending request? You will need to submit it again.",
      "empty": "No approval requests",
      "loadFailed": "Failed to load approval requests",
      "tabs": {
        "pending": "Pending",
        "processed": "Processed"
      },
      "columns": {
        "time": "Time",
        "requester": "Requester",
        "action": "Request",
        "target": "Target",
        "status": "Status",
        "decision": "Outcome",
        "actions": "Actions"
      },
      "decision": {
        "approvedBy": "Approved by",
        "rejectedBy": "Rejected by",
        "cancelledBy": "Withdrawn by",
        "reason": "Reason",
        "noReason": "No reason given",
        "result": "Execution",
        "executedOk": "Succeeded (HTTP {code}) · {time}",
        "executedFailed": "Failed: {error} (HTTP {code})",
        "expired": "Expired at",
        "expiresAt": "Valid until {time}"
      },
      "status": {
        "pending": "Pending",
        "executing": "Executing",
        "approved": "Approved",
        "failed": "Failed",
        "rejected": "Rejected",
        "cancelled": "Withdrawn",
        "expired": "Expired"
      },
      "actions": {
        "approve": "Approve",
        "approveSelected": "Approve selected ({count})",
        "approveAll": "Approve all pending",
        "reject": "Reject",
        "cancel": "Withdraw",
        "detail": "Details"
      },
      "batchConfirm": "Approve {count} pending request(s)? They execute immediately, one by one, under your identity.",
      "batchResult": "Batch approval finished: {approved} approved, {failed} failed, {skipped} skipped",
      "batchEmpty": "No pending requests to approve",
      "batchLimitHint": "Up to {count} per batch; larger sets run in several batches (adjustable in system settings)",
      "detail": {
        "title": "Request details",
        "titleWithId": "Request #{id}",
        "payload": "Request body (redacted)",
        "decision": "Decision",
        "result": "Execution result",
        "statusCode": "HTTP status",
        "expiresAt": "Expires at",
        "close": "Close"
      },
      "describe": {
        "fields": {
          "email": "Email",
          "username": "Username",
          "notes": "Notes",
          "balance": "Balance",
          "initialBalance": "Initial balance",
          "concurrency": "Concurrency",
          "rpmLimit": "RPM limit",
          "allowedGroups": "Allowed groups",
          "restrictPublicGroups": "Restrict public groups",
          "resetPassword": "Reset password",
          "status": "Status",
          "groupRates": "Group rate overrides",
          "role": "Role",
          "group": "Group",
          "users": "Users",
          "validity": "Validity",
          "providerKey": "Provider key",
          "providerSubject": "Provider subject",
          "resetRateLimit": "Reset rate-limit usage",
          "messageTitle": "Message title",
          "messageContent": "Message content"
        },
        "values": {
          "yes": "Yes",
          "no": "No",
          "none": "None",
          "remove": "remove",
          "redacted": "(redacted)",
          "statusActive": "Active",
          "statusDisabled": "Disabled",
          "unlimited": "Unlimited",
          "quotaDaily": "Daily",
          "quotaWeekly": "Weekly",
          "quotaMonthly": "Monthly",
          "allWindows": "all",
          "scopeAll": "all users",
          "scopeUsers": "{count} users ({ids})",
          "days": "{n} days",
          "validityDefault": "Group default",
          "bodyTooLong": "Content too long to preview. Expand the raw request below."
        },
        "summary": {
          "userCreate": "Create user {email}",
          "userUpdate": "Update profile of {target}",
          "balanceAdd": "Add {amount} balance to {target}",
          "balanceSubtract": "Deduct {amount} balance from {target}",
          "balanceSet": "Set balance of {target} to {amount}",
          "replaceGroup": "Replace exclusive group of {target}: {from} to {to}",
          "batchConcurrencySet": "Set concurrency of {scope} to {value}",
          "batchConcurrencyAdd": "Increase concurrency of {scope} by {value}",
          "batchLimits": "Batch update limits of {scope}",
          "platformQuotas": "Update platform quotas of {target}",
          "platformQuotaReset": "Reset the {window} quota window of {target} on {platform}",
          "attributes": "Update attributes of {target}",
          "authIdentity": "Bind {provider} identity to {target}",
          "apiKeyBind": "Bind API key {target} to group {group}",
          "apiKeyUnbind": "Unbind group from API key {target}",
          "apiKeyUpdate": "Update API key {target}",
          "subscriptionAssign": "Assign subscription: {target}",
          "subscriptionBulkAssign": "Assign subscription of group {group} to {count} users",
          "subscriptionExtend": "Extend subscription {target} by {days} days",
          "subscriptionShorten": "Shorten subscription {target} by {days} days",
          "subscriptionResetQuota": "Reset {windows} quota of subscription {target}",
          "subscriptionRevoke": "Revoke subscription {target}",
          "subscriptionRestore": "Restore subscription {target}",
          "subscriptionDelete": "Delete subscription {target}",
          "userSiteMessage": "Send a site message to {target}",
          "unknown": "{action}: {target}"
        }
      },
      "rawToggle": "Raw request (redacted)",
      "noFields": "No extra fields",
      "actionLabels": {
        "userCreate": "Create user",
        "userUpdate": "Edit user",
        "userBalance": "Adjust balance",
        "userReplaceGroup": "Replace exclusive group",
        "userBatchConcurrency": "Batch update concurrency",
        "userBatchLimits": "Batch update limits",
        "userPlatformQuotas": "Update platform quotas",
        "userPlatformQuotaReset": "Reset platform quota window",
        "userAttributes": "Update user attributes",
        "userAuthIdentity": "Bind login identity",
        "apiKeyUpdate": "Change API key group",
        "subscriptionAssign": "Assign subscription",
        "subscriptionBulkAssign": "Bulk assign subscriptions",
        "subscriptionExtend": "Adjust subscription validity",
        "subscriptionResetQuota": "Reset subscription quota",
        "subscriptionRevoke": "Revoke subscription",
        "subscriptionRestore": "Restore subscription",
        "userSiteMessage": "Send site message"
      }
    }
  },
  "tickets": {
    "title": "Support Tickets",
    "description": "Report a problem and talk to support",
    "caption": "Support Tickets",
    "intro": "Having trouble with your account, billing or API calls? Open a ticket and support will reply in the same thread.",
    "create": "New ticket",
    "createTitle": "New ticket",
    "empty": "No tickets yet",
    "loading": "Loading...",
    "loadFailed": "Failed to load tickets",
    "form": {
      "title": "Title",
      "titlePlaceholder": "Summarize the problem in one line",
      "category": "Category",
      "body": "Description",
      "bodyPlaceholder": "Describe what happened: symptoms, the model / API key involved, error messages, when it started. Markdown is supported.",
      "bodyHint": "Markdown supported, up to {max} characters",
      "submit": "Submit ticket",
      "submitting": "Submitting..."
    },
    "status": {
      "open": "Open",
      "replied": "Replied",
      "closed": "Closed"
    },
    "statusFilter": {
      "all": "All"
    },
    "category": {
      "account": "Account",
      "billing": "Billing / Quota",
      "api": "API calls",
      "other": "Other",
      "appeal": "Account appeal"
    },
    "columns": {
      "id": "ID",
      "user": "User",
      "title": "Title",
      "category": "Category",
      "status": "Status",
      "messages": "Messages",
      "lastMessage": "Last updated",
      "actions": "Actions"
    },
    "thread": {
      "you": "You",
      "user": "User",
      "staff": "Support",
      "roleAdmin": "Admin",
      "roleOperator": "Operator",
      "placeholder": "Write a reply, Markdown supported",
      "send": "Send",
      "sending": "Sending...",
      "closedHint": "This ticket is closed. Reopen it to continue the conversation.",
      "unread": "New reply"
    },
    "attachments": {
      "insertImage": "Insert image",
      "hint": "Paste, drop or pick an image, up to 5 MB each",
      "uploading": "Uploading image...",
      "uploadFailed": "Image upload failed",
      "errors": {
        "storageNotConfigured": "Ticket attachment storage is not configured. Please contact an administrator.",
        "tooLarge": "Image exceeds the size limit (max 5 MB each)",
        "badType": "Only image files are supported"
      }
    },
    "actions": {
      "close": "Close ticket",
      "reopen": "Reopen",
      "view": "View",
      "dismiss": "Dismiss"
    },
    "confirmClose": {
      "title": "Close ticket",
      "message": "Replies are disabled once the ticket is closed. You can reopen it later if needed. Close it now?"
    },
    "toast": {
      "created": "Ticket submitted",
      "replied": "Reply sent",
      "closed": "Ticket closed",
      "reopened": "Ticket reopened"
    },
    "errors": {
      "openLimit": "Too many open tickets. Please close or resolve an existing one first.",
      "closed": "This ticket is closed",
      "notClosed": "This ticket is not closed"
    },
    "detail": {
      "title": "Ticket",
      "titleWithId": "Ticket #{id}"
    },
    "admin": {
      "title": "Tickets",
      "description": "Review and reply to user tickets",
      "empty": "No tickets match the current filters",
      "searchPlaceholder": "Search by title or user email",
      "categoryAll": "All categories",
      "userStatus": {
        "active": "Account active",
        "disabled": "Account disabled"
      },
      "restoreAccount": "Restore account",
      "restoreConfirmTitle": "Restore account",
      "restoreConfirmMessage": "Restore the account of {email}? They will be able to sign in and call the API again, and will receive a site message.",
      "restoreSuccess": "Account restored",
      "tabs": {
        "open": "Open",
        "replied": "Replied",
        "closed": "Closed",
        "all": "All"
      }
    }
  },
  "appeal": {
    "title": "Account appeal",
    "exit": "Exit",
    "banner": {
      "title": "Your account has been disabled",
      "desc": "See why it was disabled and submit an appeal here. An administrator will reply in the ticket. Once the account is restored, sign in again."
    },
    "expiresAt": "This appeal session is valid until {time}",
    "notices": "Notices",
    "noNotices": "No notices",
    "ticketTitle": "Appeal",
    "createTitle": "Explain the situation and submit an appeal. Each account can have one open appeal at a time.",
    "submitAgain": "Submit a new appeal",
    "form": {
      "title": "Title",
      "defaultTitle": "Request to restore my account",
      "body": "Details",
      "bodyPlaceholder": "Describe what the account is used for, why the risk rule may have been triggered, and how you will avoid it",
      "submit": "Submit appeal",
      "submitted": "Appeal submitted. Please wait for an administrator to respond."
    },
    "errors": {
      "activeExists": "You already have an open appeal. Continue the conversation below.",
      "rateLimited": "Too many requests. Please try again later.",
      "expired": "The appeal session has expired. Please sign in again."
    },
    "loginNotice": {
      "restored": "Your account has been restored. Please sign in again.",
      "expired": "The appeal session has expired. Sign in again to continue your appeal."
    }
  },
  "siteMessages": {
    "title": "Messages",
    "open": "Open messages",
    "tabs": {
      "all": "All",
      "unread": "Unread"
    },
    "empty": "No messages yet",
    "emptyUnread": "No unread messages",
    "loadMore": "Load more",
    "loadFailed": "Failed to load messages",
    "markAllRead": "Mark all as read",
    "allMarkedRead": "All messages marked as read",
    "back": "Close",
    "from": {
      "system": "System",
      "staff": "Staff"
    },
    "categories": {
      "security": "Security",
      "admin": "Notice",
      "system": "System"
    },
    "popup": {
      "badge": "New message",
      "acknowledge": "Got it",
      "viewAll": "View all",
      "more": "{count} more unread"
    }
  }
} as const
