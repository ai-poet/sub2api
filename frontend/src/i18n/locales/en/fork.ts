// fork 自有的翻译键（上游 locale 中不存在）。
// 单独成文件：同步上游 locale 模块时不会与之冲突，index.ts 里深合并生效。
export default {
  "common": {
    "retry": "Retry"
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
      "titleAccent": "",
      "titleTail": "",
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
    "proofStrip": {
      "overline": "Platform capabilities"
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
    "value": {
      "overline": "Why pick us",
      "title": "Make running Claude Code and Codex feel effortless",
      "description": "Not another IDE. Not another API relay. We stitch sign-up, top-up, tool configuration, usage visibility, and failure diagnosis into one smooth path.",
      "items": {
        "economics": {
          "eyebrow": "01 / Fast setup",
          "title": "One-tap local configuration",
          "description": "Sign in on the desktop client and it configures Claude Code and Codex for you — no manual JSON edits, and existing settings stay intact."
        },
        "reliability": {
          "eyebrow": "02 / Clear spend",
          "title": "One balance panel covers both surfaces",
          "description": "How much each of Claude and Codex used, how much quota is left, current rate-limit status — all on one screen, no more hopping between platforms."
        },
        "control": {
          "eyebrow": "03 / All in one",
          "title": "SLA-backed service",
          "description": "Availability, latency, and pricing are managed end to end. When something breaks, the platform auto-fails over or gives clear recovery guidance."
        }
      }
    },
    "comparison": {
      "overline": "A different angle",
      "description": ""
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
    "trust": {
      "overline": "Use with confidence",
      "title": "See where every dollar goes and why things break",
      "description": "The most common pain with relay services is opacity: no idea about balance, no idea why a call fails. We surface these facts so you do not have to guess.",
      "cards": {
        "gateway": {
          "title": "Per-route accounting",
          "description": "Each route is billed independently so one failure does not poison the entire bill."
        },
        "resilience": {
          "title": "Resilient switching",
          "description": "When one upstream hits limits, jitters, or short-lived failures, the platform absorbs the switching overhead."
        },
        "visibility": {
          "title": "Failures explained",
          "description": "Clear failure reasons and recovery guidance — no black-box errors."
        }
      },
      "trackers": {
        "routing": "Sticky sessions",
        "billing": "Metered billing",
        "visibility": "Usage detail"
      }
    },
    "cta": {
      "eyebrow": "READY TO START",
      "stat": "One gateway · Metered billing · Built for daily development"
    }
  },
  "changelog": {
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
    "latestLatency": "Latest Latency",
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
    "avgLatency": "Avg Latency",
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
      "down": "Outage"
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
    "communityGroupJoin": "Join Now"
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
      "typeReferralReward": "Balance (Referral Reward)"
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
        "slowLatencyMs": "Slow Threshold (ms)",
        "latestResult": "Latest Result Preview",
        "latestResultHint": "Shows the latest stable state and raw probe summary.",
        "latestResultEmpty": "No probe result yet. Save the config or run an immediate probe to populate this area.",
        "currentStatus": "Current Status",
        "checkedAt": "Checked At",
        "latency": "Latency",
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
        "footerHint": "\"Probe now\" saves the current form first and then runs a probe immediately."
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
        "publicHint": "Use a public http(s) link from object storage, a CDN, or a release platform. Custom home page content still fully controls the home page when configured."
      },
      "changelog": {
        "title": "Changelog",
        "description": "Manage client version changelog entries. Enabled entries will be shown on the /changelog page.",
        "addEntry": "Add Entry",
        "version": "Version",
        "versionPlaceholder": "e.g. 1.0.0",
        "publishedAt": "Published Date",
        "titleLabel": "Title",
        "titlePlaceholder": "Update title",
        "items": "Items",
        "itemPlaceholder": "Supports Markdown",
        "addItem": "Add Item",
        "enabled": "Enabled",
        "delete": "Delete",
        "deleteConfirm": "Are you sure you want to delete this entry?",
        "emptyHint": "No changelog entries yet. Click \"Add Entry\" to create one.",
        "preview": "Preview",
        "edit": "Edit",
        "moveUp": "Move Up",
        "moveDown": "Move Down"
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
      "image": "Per image"
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
      "perImage": "Per image"
    },
    "units": {
      "perMillionTokens": "Per 1M tokens",
      "perRequest": "Per request",
      "perImage": "Per image"
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
      "userRateOverride": "User override"
    },
    "expandDetails": "Expand tiers and peer groups",
    "collapseDetails": "Collapse details",
    "intervalSectionTitle": "Channel tier pricing",
    "intervalDefaultLabel": "Default tier",
    "otherGroupsTitle": "Other accessible groups for this model",
    "peerDisplayedPrice": "Displayed price for this group"
  },
  "announcements": {
    "newAnnouncement": "New Announcement"
  }
} as const
