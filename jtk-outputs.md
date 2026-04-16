## me

Show information about the currently authenticated Jira user.

### Default (table)

```
Account ID: 60e09bae7fcd820073089249
Display Name: Rian Stockbower
Email: rian@monitapp.io
Active: yes
```

### `-o json`

```json
{
  "accountId": "60e09bae7fcd820073089249",
  "displayName": "Rian Stockbower"
}
```

### `-o plain` (account ID only)

```
60e09bae7fcd820073089249
```

### Equivalent API call

```bash
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/myself"
```
## users

Search and look up Jira users.

### search

#### Default (table)

```
ACCOUNT ID | NAME | EMAIL | ACTIVE
60e09bae7fcd820073089249 | Rian Stockbower | rian@monitapp.io | yes
```

#### `-o json`

```json
{
  "results": [
    {
      "accountId": "60e09bae7fcd820073089249",
      "displayName": "Rian Stockbower"
    }
  ],
  "_meta": {
    "count": 1,
    "hasMore": false
  }
}
```

#### No results

```
No users found matching 'xyznonexistent999'
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/user/search?query=Rian"
```

### get

#### Default (table)

```
Account ID: 60e09bae7fcd820073089249
Display Name: Rian Stockbower
Email: rian@monitapp.io
Active: yes
```

#### `-o json`

```json
{
  "accountId": "60e09bae7fcd820073089249",
  "displayName": "Rian Stockbower"
}
```

#### 404 error

```
getting user 000000000000000000000000: resource not found: Specified user does not exist or you do not have required permissions
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/user?accountId=60e09bae7fcd820073089249"
```
## projects

Manage Jira projects.

### list

#### Default (table)

```
KEY | NAME | TYPE | LEAD
INCIDENT | Incidents | software | 
JAR | Jira Application Requests | software | 
MON | Platform Development | software | 
OFF | On/Offboarding | software | 
ON | Customer Onboarding | software | 
```

#### `-o json`

```json
[
  {
    "id": 10026,
    "key": "INCIDENT",
    "name": "Incidents",
    "projectTypeKey": "software"
  },
  {
    "id": 10024,
    "key": "JAR",
    "name": "Jira Application Requests",
    "projectTypeKey": "software"
  },
  {
    "id": 10022,
    "key": "MON",
    "name": "Platform Development",
    "projectTypeKey": "software"
  },
  {
    "id": 10025,
    "key": "OFF",
    "name": "On/Offboarding",
    "projectTypeKey": "software"
  },
  {
    "id": 10023,
    "key": "ON",
    "name": "Customer Onboarding",
    "projectTypeKey": "software"
  }
]
```

#### `--query` filter

```
KEY | NAME | TYPE | LEAD
MON | Platform Development | software | 
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/project/search?maxResults=5"
# With query: ?query=Platform
```

### get

#### Default (table)

```
Key: MON
Name: Platform Development
ID: 10022
Type: software
Lead: Rian Stockbower
Issue Types: [Epic Kanban SDLC]
```

#### `-o json`

```json
{
  "id": 10022,
  "key": "MON",
  "name": "Platform Development",
  "projectTypeKey": "software",
  "lead": {
    "accountId": "60e09bae7fcd820073089249",
    "displayName": "Rian Stockbower",
    "active": true,
    "avatarUrls": {
      "16x16": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/60e09bae7fcd820073089249/724b7268-43a2-4a44-a00e-7a204fe99f90/16",
      "24x24": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/60e09bae7fcd820073089249/724b7268-43a2-4a44-a00e-7a204fe99f90/24",
      "32x32": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/60e09bae7fcd820073089249/724b7268-43a2-4a44-a00e-7a204fe99f90/32",
      "48x48": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/60e09bae7fcd820073089249/724b7268-43a2-4a44-a00e-7a204fe99f90/48"
    }
  },
  "issueTypes": [
    {
      "id": "10000",
      "name": "Epic",
      "description": "A big user story that needs to be broken down. Created by Jira Software - do not edit or delete.",
      "subtask": false
    },
    {
      "id": "10026",
      "name": "Kanban",
      "description": "Task following Kanban Flow",
      "subtask": false
    },
    {
      "id": "10025",
      "name": "SDLC",
      "description": "Task requiring Software Development Life Cycle",
      "subtask": false
    }
  ],
  "components": [
    {
      "id": "10143",
      "name": "Admin Portal"
    },
    {
      "id": "10144",
      "name": "Admin Service"
    },
    {
      "id": "10145",
      "name": "Banker Portal"
    },
    {
      "id": "10147",
      "name": "Codat Sync Service"
    },
    {
      "id": "10146",
      "name": "Config Service"
    },
    {
      "id": "10148",
      "name": "Data Classification"
    },
    {
      "id": "10149",
      "name": "Document Management Service"
    },
    {
      "id": "10163",
      "name": "FI Onboarding"
    },
    {
      "id": "10150",
      "name": "Identity and Access Management Service"
    },
    {
      "id": "10151",
      "name": "Insights Engine"
    },
    {
      "id": "10152",
      "name": "Insights Service"
    },
    {
      "id": "10153",
      "name": "Integration - Apiture"
    },
    {
      "id": "10154",
      "name": "Integration - Banno"
    },
    {
      "id": "10155",
      "name": "Integration - Narmi"
    },
    {
      "id": "10156",
      "name": "Integration - Q2"
    },
    {
      "id": "10157",
      "name": "Ledger Data Ingester"
    },
    {
      "id": "10176",
      "name": "Monitoring"
    },
    {
      "id": "10181",
      "name": "Next Best Action Service"
    },
    {
      "id": "10158",
      "name": "Notification Service"
    },
    {
      "id": "10182",
      "name": "Quant Database"
    },
    {
      "id": "10159",
      "name": "Signal Service"
    },
    {
      "id": "10160",
      "name": "Tenant Management Service"
    },
    {
      "id": "10178",
      "name": "User Authorization"
    },
    {
      "id": "10161",
      "name": "User Management Service"
    },
    {
      "id": "10162",
      "name": "User Portal"
    }
  ]
}
```

#### 404 error

```
fetching project: resource not found: No project could be found with key 'NONEXISTENT'.
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/project/MON"
```

### types

#### Default (table)

```
KEY | FORMATTED
product_discovery | Product Discovery
software | Software
service_desk | Service Desk
customer_service | Customer Service
business | Business
```

#### `-o json`

```json
[
  {
    "key": "product_discovery",
    "formattedKey": "Product Discovery",
    "descriptionI18nKey": "jira.project.type.polaris.description"
  },
  {
    "key": "software",
    "formattedKey": "Software",
    "descriptionI18nKey": "jira.project.type.software.description"
  },
  {
    "key": "service_desk",
    "formattedKey": "Service Desk",
    "descriptionI18nKey": "jira.project.type.servicedesk.description.jsm"
  },
  {
    "key": "customer_service",
    "formattedKey": "Customer Service",
    "descriptionI18nKey": "jcs.project.type.customer.service.description"
  },
  {
    "key": "business",
    "formattedKey": "Business",
    "descriptionI18nKey": "jira.project.type.business.description"
  }
]
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/project/type"
```
## issues

Manage Jira issues.

### list

#### Default (table)

```
KEY | SUMMARY | STATUS | ASSIGNEE | TYPE
MON-4810 | Audit and remediate accessibility issues on Cap... | In Code Review | Aaron Wong | SDLC
MON-4807 | Make CapOne key-stack authoritative for zero-st... | In Code Review | Aaron Wong | SDLC
MON-4809 | Bump PostHog sampling to 100% for CapOne sessions | Backlog | Unassigned | SDLC
More results available (use --next-page-token to fetch next page)
```

#### `-o json`

```json
{
  "results": [
    {
      "key": "MON-4810",
      "summary": "Audit and remediate accessibility issues on CapOne-specific surfaces",
      "status": "In Code Review",
      "type": "SDLC",
      "assignee": "Aaron Wong"
    },
    {
      "key": "MON-4807",
      "summary": "Make CapOne key-stack authoritative for zero-state back behavior",
      "status": "In Code Review",
      "type": "SDLC",
      "assignee": "Aaron Wong"
    },
    {
      "key": "MON-4809",
      "summary": "Bump PostHog sampling to 100% for CapOne sessions",
      "status": "Backlog",
      "type": "SDLC"
    }
  ],
  "_meta": {
    "count": 3,
    "hasMore": true
  }
}
```

#### `-o plain`

```
KEY | SUMMARY | STATUS | ASSIGNEE | TYPE
MON-4810 | Audit and remediate accessibility issues on Cap... | In Code Review | Aaron Wong | SDLC
MON-4807 | Make CapOne key-stack authoritative for zero-st... | In Code Review | Aaron Wong | SDLC
MON-4809 | Bump PostHog sampling to 100% for CapOne sessions | Backlog | Unassigned | SDLC
More results available (use --next-page-token to fetch next page)
```

#### `--sprint current`

```
KEY | SUMMARY | STATUS | ASSIGNEE | TYPE
MON-4810 | Audit and remediate accessibility issues on Cap... | In Code Review | Aaron Wong | SDLC
MON-4807 | Make CapOne key-stack authoritative for zero-st... | In Code Review | Aaron Wong | SDLC
MON-4757 | Prototype html-in-canvas chart engine to replac... | In Code Review | Aaron Wong | SDLC
More results available (use --next-page-token to fetch next page)
```

#### `--all-fields -o json`

```json
{
  "results": [
    {
      "key": "MON-4810",
      "summary": "Audit and remediate accessibility issues on CapOne-specific surfaces",
      "status": "In Code Review",
      "type": "SDLC",
      "assignee": "Aaron Wong"
    },
    {
      "key": "MON-4807",
      "summary": "Make CapOne key-stack authoritative for zero-state back behavior",
      "status": "In Code Review",
      "type": "SDLC",
      "assignee": "Aaron Wong"
    },
    {
      "key": "MON-4809",
      "summary": "Bump PostHog sampling to 100% for CapOne sessions",
      "status": "Backlog",
      "type": "SDLC"
    }
  ],
  "_meta": {
    "count": 3,
    "hasMore": true
  }
}
```

#### `--fields summary,status -o json`

```json
{
  "results": [
    {
      "key": "MON-4810",
      "summary": "Audit and remediate accessibility issues on CapOne-specific surfaces",
      "status": "In Code Review"
    },
    {
      "key": "MON-4807",
      "summary": "Make CapOne key-stack authoritative for zero-state back behavior",
      "status": "In Code Review"
    },
    {
      "key": "MON-4809",
      "summary": "Bump PostHog sampling to 100% for CapOne sessions",
      "status": "Backlog"
    }
  ],
  "_meta": {
    "count": 3,
    "hasMore": true
  }
}
```

#### Project not found

```
No issues found
```

#### Equivalent API call

```bash
# issues list uses POST /search/jql (not GET /search)
curl -u EMAIL:TOKEN -X POST "https://monitproduct.atlassian.net/rest/api/3/search/jql" \
  -H "Content-Type: application/json" \
  -d '{"jql":"project = MON","maxResults":3}'
# Note: auto-paginates for --max > 100 (multiple calls)
```

### get

#### Default (table)

```
Key: MON-4810
Summary: Audit and remediate accessibility issues on CapOne-specific surfaces
Status: In Code Review
Type: SDLC
Priority: Medium
Assignee: Aaron Wong
Project: MON
Description: 
Summary
Perform an accessibility-focused review and remediation pass across CapOne-specific frontend surfaces in packages/legacy/app, then validate the highest-risk interaction patterns.
Primary audi... [truncated, use --no-truncate for complete text]
URL: https://monitproduct.atlassian.net/browse/MON-4810
```

#### `-o json`

```json
{
  "key": "MON-4810",
  "summary": "Audit and remediate accessibility issues on CapOne-specific surfaces",
  "status": "In Code Review",
  "type": "SDLC",
  "assignee": "Aaron Wong"
}
```

#### `--no-truncate`

```
Key: MON-4810
Summary: Audit and remediate accessibility issues on CapOne-specific surfaces
Status: In Code Review
Type: SDLC
Priority: Medium
Assignee: Aaron Wong
Project: MON
Description: 
Summary
Perform an accessibility-focused review and remediation pass across CapOne-specific frontend surfaces in packages/legacy/app, then validate the highest-risk interaction patterns.
Primary audit artifact:
- docs/capone-accessibility-audit-2026-04-15.md


Problem
The CapOne-specific UI surfaces are in mixed shape from an accessibility standpoint. Source review found a few meaningful issues concentrated in loading/redirect status communication, interactive semantics, and modal/illustration behavior.
The biggest findings are:
- loading / redirect surfaces do not expose accessible status updates
- CaponeStepsPreview uses tooltip semantics for interactive content
- CaponeUnsupportedPackageModal is missing aria-describedby
- some splash / package-option images are likely decorative but may be over-announced
- CapOne-specific a11y coverage is currently light for keyboard / SR-focused behavior


In Scope
CapOne-specific surfaces including:
- packages/legacy/app/caponeLoadingScreen.ts
- packages/legacy/app/containers/CaponeRedirect/CaponeRedirect.tsx
- packages/legacy/app/fi-experiences/capone/CaponeTopBanner.tsx
- packages/legacy/app/fi-experiences/capone/CaponeErrorPage.tsx
- packages/legacy/app/fi-experiences/capone/CaponeSmbSplash.tsx
- packages/legacy/app/fi-experiences/capone/CaponeStepsPreview.tsx
- packages/legacy/app/fi-experiences/capone/CaponeUnsupportedPackageModal.tsx
- packages/legacy/app/fi-experiences/capone/CaponeAccountingPackageOption.tsx


Proposed Work

1. Loading / redirect accessibility
- add accessible loading/status messaging to the CapOne loading screen and redirect/loading surfaces
- ensure users of assistive tech receive meaningful progress/state feedback during SSO transfer and redirect waiting states


2. Interactive semantics cleanup
- replace tooltip semantics in the CapOne steps preview flow with a more appropriate interactive pattern for content that contains controls
- improve trigger labeling/state where needed


3. Modal relationship fix
- add aria-describedby support for the unsupported-package modal body content


4. Decorative / redundant image announcement review
- review CapOne splash imagery and package-option logos
- mark decorative images as decorative where appropriate
- avoid duplicate accessible naming where visible text already names the option/content


5. Validation / coverage
- add or update targeted tests for the highest-confidence fixes
- run manual keyboard + screen-reader-focused validation on the main CapOne surfaces


Acceptance Criteria
- CapOne loading and redirect states expose meaningful accessible status to assistive technologies
- CaponeStepsPreview no longer exposes interactive content as a tooltip-style surface
- the unsupported-package modal exposes both title and body text correctly to assistive tech
- decorative CapOne imagery is not unnecessarily announced
- package-option controls do not produce redundant or confusing accessible names
- CapOne-specific keyboard and screen-reader validation is documented for the remediated surfaces


Notes
This ticket intentionally keeps all CapOne accessibility findings in one place for now, rather than splitting into multiple implementation tickets.

URL: https://monitproduct.atlassian.net/browse/MON-4810
```

#### 404 error

```
fetching issue: resource not found: Issue does not exist or you do not have permission to see it.
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/issue/MON-4810"
```

### search

#### Default (table)

```
KEY | SUMMARY | STATUS | ASSIGNEE | TYPE
MON-4810 | Audit and remediate accessibility issues on Cap... | In Code Review | Aaron Wong | SDLC
MON-4809 | Bump PostHog sampling to 100% for CapOne sessions | Backlog | Unassigned | SDLC
MON-4808 | Support deep-link tab navigation in Q2 campaign... | Backlog | Unassigned | SDLC
More results available (use --next-page-token to fetch next page)
```

#### `-o json`

```json
{
  "results": [
    {
      "key": "MON-4810",
      "summary": "Audit and remediate accessibility issues on CapOne-specific surfaces",
      "status": "In Code Review",
      "type": "SDLC",
      "assignee": "Aaron Wong"
    },
    {
      "key": "MON-4809",
      "summary": "Bump PostHog sampling to 100% for CapOne sessions",
      "status": "Backlog",
      "type": "SDLC"
    },
    {
      "key": "MON-4808",
      "summary": "Support deep-link tab navigation in Q2 campaign CTA URLs",
      "status": "Backlog",
      "type": "SDLC"
    }
  ],
  "_meta": {
    "count": 3,
    "hasMore": true
  }
}
```

#### No results

```
No issues found
```

#### Bad JQL

```
searching issues: bad request: Error in the JQL Query: Expecting operator but got 'jql'. The valid operators are '=', '!=', '<', '>', '<=', '>=', '~', '!~', 'IN', 'NOT IN', 'IS' and 'IS NOT'. (line 1, character 9)
```

#### Equivalent API call

```bash
# Uses POST /search/jql (not GET /search)
curl -u EMAIL:TOKEN -X POST "https://monitproduct.atlassian.net/rest/api/3/search/jql" \
  -H "Content-Type: application/json" \
  -d '{"jql":"project = MON","maxResults":3}'
```

### types

#### Default (table)

```
ID | NAME | SUBTASK | DESCRIPTION
10000 | Epic | no | A big user story that needs to be broken down. Created by...
10026 | Kanban | no | Task following Kanban Flow
10025 | SDLC | no | Task requiring Software Development Life Cycle
```

#### `-o json`

```json
[
  {
    "id": "10000",
    "name": "Epic",
    "description": "A big user story that needs to be broken down. Created by Jira Software - do not edit or delete.",
    "subtask": false
  },
  {
    "id": "10026",
    "name": "Kanban",
    "description": "Task following Kanban Flow",
    "subtask": false
  },
  {
    "id": "10025",
    "name": "SDLC",
    "description": "Task requiring Software Development Life Cycle",
    "subtask": false
  }
]
```

#### 404 error

```
fetching project: resource not found: No project could be found with key 'NONEXISTENT'.
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/issuetype/project?projectId=10022"
```

### fields

#### Default (table, truncated)

```
ID | NAME | TYPE | CUSTOM
statuscategorychangedate | Status Category Changed | datetime | no
fixVersions | Fix versions | array | no
statusCategory | Status Category | statusCategory | no
parent | Parent |  | no
resolution | Resolution | resolution | no
lastViewed | Last Viewed | datetime | no
priority | Priority | priority | no
customfield_10189 | Product Display Name | string | yes
labels | Labels | array | no
timeestimate | Remaining Estimate | number | no
aggregatetimeoriginalestimate | Σ Original Estimate | number | no
versions | Affects versions | array | no
issuelinks | Linked Issues | array | no
assignee | Assignee | user | no
status | Status | status | no
components | Components | array | no
issuekey | Key |  | no
customfield_10050 | Online Banking URL | string | yes
customfield_10051 | Design | array | yes
```

#### `--custom`

```
ID | NAME | TYPE | CUSTOM
customfield_10189 | Product Display Name | string | yes
customfield_10050 | Online Banking URL | string | yes
customfield_10051 | Design | array | yes
customfield_10052 | Vulnerability | any | yes
customfield_10053 | Sentiment | array | yes
customfield_10054 | Goals | array | yes
customfield_10055 | Focus Areas | array | yes
customfield_10049 | Migration Archive | string | yes
customfield_10040 | Send Gainsight Emails | option | yes
customfield_10041 | Send MSR Emails | option | yes
customfield_10043 | Category | option | yes
customfield_10044 | Meta Status | array | yes
customfield_10046 | QA Notes | string | yes
customfield_10039 | Theming & Branding Info | string | yes
customfield_10030 | Total forms | number | yes
customfield_10031 | Project overview key | string | yes
customfield_10032 | Project overview status | string | yes
customfield_10154 | Products | array | yes
customfield_10155 | PPLambdaUrl | string | yes
```

#### With issue key (editable fields for specific issue)

```
ID | NAME | TYPE | REQUIRED
attachment | Attachment | array | no
reporter | Reporter | user | yes
description | Description | string | no
resolution | Resolution | resolution | no
customfield_10036 | Bank Name | string | no
customfield_10005 | Change type | option | no
customfield_10049 | Migration Archive | string | no
summary | Summary | string | yes
duedate | Due date | date | no
parent | Parent | issuelink | no
issuetype | Issue Type | issuetype | yes
components | Components | array | no
customfield_10020 | Sprint | array | no
customfield_10044 | Meta Status | array | no
priority | Priority | priority | no
customfield_10018 | Parent Link | any | no
environment | Environment | string | no
customfield_10035 | Story Points | number | no
customfield_10014 | Epic Link | any | no
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/field"
# With issue context: /issue/MON-4810/editmeta
```

### field-options

#### Default (table)

```
Allowed values for field 'Priority':
ID | VALUE
1 | Highest
2 | High
3 | Medium
4 | Low
5 | Lowest
```

#### `-o json`

```json
[
  {
    "id": "1",
    "name": "Highest"
  },
  {
    "id": "2",
    "name": "High"
  },
  {
    "id": "3",
    "name": "Medium"
  },
  {
    "id": "4",
    "name": "Low"
  },
  {
    "id": "5",
    "name": "Lowest"
  }
]
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/issue/MON-4810/editmeta" | jq '.fields.priority.allowedValues'
```

### move

Moves one or more issues to a different project or type. The operation is asynchronous; by default jtk waits for completion.

**Flags:** `--to-project` (required), `--to-type` (optional, auto-detected), `--notify` (default true), `--wait` (default true, set `--wait=false` to return immediately with a task ID)

#### Default (--wait, table)

```
Moving 1 issue(s) to MON (Kanban)...
Waiting for move to complete...
Moved 1 issue(s) to MON
```

#### `--wait=false` (returns task ID immediately)

```
Moving 1 issue(s) to MON (SDLC)...
Move initiated (Task ID: 68325)
Check status with: jtk issues move-status 68325
```

#### Error: unknown target project

```
getting target project issue types: fetching project issue types: resource not found: No project could be found with key 'DOESNOTEXIST'.
```

#### Equivalent API calls

```bash
# 1. Resolve target issue type ID
curl -u EMAIL:TOKEN "BASE/rest/api/3/project/MON" | jq '.issueTypes[] | select(.name=="Kanban") | .id'

# 2. Initiate move (async)
curl -u EMAIL:TOKEN -X POST "BASE/rest/api/3/bulk/issues/move" \
  -H "Content-Type: application/json" \
  -d '{"sendBulkNotification":true,"targetToSourcesMapping":{"MON,<issueTypeId>":{"issueIdsOrKeys":["MON-4816"],"inferFieldDefaults":true,"inferStatusDefaults":true}}}'

# 3. Poll task status
curl -u EMAIL:TOKEN "BASE/rest/api/3/bulk/queue/<taskId>"
```

### move-status

Polls the status of an async move task returned by `issues move --wait=false`. Single call — does not loop.

**Args:** `<task-id>` (required)

#### Default (table)

```
Task ID: 68325
Status: COMPLETE
Progress: 0%
Submitted: 
```

#### `-o json`

```json
{
  "taskId": "68325",
  "status": "COMPLETE",
  "submittedAt": "",
  "progress": 0
}
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN "BASE/rest/api/3/bulk/queue/<taskId>"
```
## boards

Manage agile boards.

### list

#### Default (table)

```
ID | NAME | TYPE | PROJECT
26 | OFF board | simple | OFF
27 | INCIDENT board | simple | INCIDENT
24 | ON board | kanban | ON
23 | MON board | scrum | MON
25 | JAR board | kanban | JAR
12 | OP board | kanban | OP
28 | TST board | scrum | 
```

#### `-o json`

```json
{
  "results": [
    {
      "id": 26,
      "name": "OFF board",
      "type": "simple"
    },
    {
      "id": 27,
      "name": "INCIDENT board",
      "type": "simple"
    },
    {
      "id": 24,
      "name": "ON board",
      "type": "kanban"
    },
    {
      "id": 23,
      "name": "MON board",
      "type": "scrum"
    },
    {
      "id": 25,
      "name": "JAR board",
      "type": "kanban"
    },
    {
      "id": 12,
      "name": "OP board",
      "type": "kanban"
    },
    {
      "id": 28,
      "name": "TST board",
      "type": "scrum"
    }
  ],
  "_meta": {
    "count": 7,
    "hasMore": false
  }
}
```

#### `-p MON` (filtered by project)

```
ID | NAME | TYPE | PROJECT
24 | ON board | kanban | ON
23 | MON board | scrum | MON
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/agile/1.0/board?maxResults=50"
# With project filter: ?projectKeyOrId=MON
```

### get

#### Default (table)

```
ID: 23
Name: MON board
Type: scrum
Project: MON
```

#### `-o json`

```json
{
  "id": 23,
  "name": "MON board",
  "type": "scrum"
}
```

#### 404 error

```
getting board 99999: resource not found: rapidViewId: The requested board cannot be viewed because it either does not exist or you do not have permission to view it.
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/agile/1.0/board/23"
```
## sprints

Manage sprints. (Requires Basic Auth — scoped tokens lack Agile scope.)

### list

#### Default (table)

```
ID | NAME | STATE | START | END
58 | MON Sprint 1 | closed | 2023-08-14 | 2023-08-25
59 | MON Sprint 2 | closed | 2023-08-28 | 2023-09-08
60 | MON Sprint 3 | closed | 2023-09-11 | 2023-09-22
61 | MON Sprint 4 | closed | 2023-09-25 | 2023-10-06
62 | MON Sprint 5 | closed | 2023-10-11 | 2023-10-20
63 | MON Sprint 6 | closed | 2023-10-23 | 2023-11-03
64 | MON Sprint 7 | closed | 2023-11-06 | 2023-11-17
65 | MON Sprint 8 | closed | 2023-11-20 | 2023-12-01
66 | MON Sprint 9 | closed | 2023-12-04 | 2023-12-15
67 | MON Sprint 10 | closed | 2023-12-18 | 2024-01-05
68 | MON Sprint 11 | closed | 2024-01-08 | 2024-01-19
69 | MON Sprint 12 | closed | 2024-01-22 | 2024-02-02
70 | MON Sprint 14 | closed | 2024-02-19 | 2024-03-01
71 | MON Sprint 15 | closed | 2024-03-04 | 2024-03-15
72 | MON Sprint 17 | closed | 2024-04-01 | 2024-04-12
73 | MON Sprint 18 | closed | 2024-04-15 | 2024-04-26
74 | MON Sprint 19 | closed | 2024-04-29 | 2024-05-10
75 | MON Sprint 20 | closed | 2024-05-13 | 2024-05-24
76 | MON Sprint 21 | closed | 2024-05-27 | 2024-06-07
```

#### `-o json`

```json
{
  "results": [
    {
      "id": 58,
      "name": "MON Sprint 1",
      "state": "closed"
    },
    {
      "id": 59,
      "name": "MON Sprint 2",
      "state": "closed"
    },
    {
      "id": 60,
      "name": "MON Sprint 3",
      "state": "closed"
    },
    {
      "id": 61,
      "name": "MON Sprint 4",
      "state": "closed"
    },
    {
      "id": 62,
      "name": "MON Sprint 5",
      "state": "closed"
    },
    {
      "id": 63,
      "name": "MON Sprint 6",
      "state": "closed"
    },
    {
      "id": 64,
      "name": "MON Sprint 7",
      "state": "closed"
    },
    {
      "id": 65,
      "name": "MON Sprint 8",
      "state": "closed"
    },
    {
      "id": 66,
      "name": "MON Sprint 9",
      "state": "closed"
    },
    {
      "id": 67,
      "name": "MON Sprint 10",
      "state": "closed"
    },
    {
      "id": 68,
      "name": "MON Sprint 11",
      "state": "closed"
    },
    {
      "id": 69,
      "name": "MON Sprint 12",
      "state": "closed"
    },
    {
      "id": 70,
      "name": "MON Sprint 14",
      "state": "closed"
    },
    {
      "id": 71,
      "name": "MON Sprint 15",
      "state": "closed"
    },
    {
      "id": 72,
      "name": "MON Sprint 17",
      "state": "closed"
    },
    {
      "id": 73,
      "name": "MON Sprint 18",
      "state": "closed"
    },
    {
      "id": 74,
      "name": "MON Sprint 19",
      "state": "closed"
    },
    {
      "id": 75,
      "name": "MON Sprint 20",
      "state": "closed"
    },
    {
      "id": 76,
      "name": "MON Sprint 21",
      "state": "closed"
    },
    {
      "id": 77,
      "name": "MON Sprint 22",
      "state": "closed"
    },
    {
      "id": 78,
      "name": "MON Sprint 23",
      "state": "closed"
    },
    {
      "id": 79,
      "name": "MON Sprint 24",
      "state": "closed"
    },
    {
      "id": 80,
      "name": "MON Sprint 25",
      "state": "closed"
    },
    {
      "id": 81,
      "name": "MON Sprint 26",
      "state": "closed"
    },
    {
      "id": 82,
      "name": "MON Sprint 27",
      "state": "closed"
    },
    {
      "id": 83,
      "name": "MON Sprint 28",
      "state": "closed"
    },
    {
      "id": 84,
      "name": "MON Sprint 29",
      "state": "closed"
    },
    {
      "id": 85,
      "name": "MON Sprint 30",
      "state": "closed"
    },
    {
      "id": 86,
      "name": "MON Sprint 31",
      "state": "closed"
    },
    {
      "id": 87,
      "name": "MON Sprint 32",
      "state": "closed"
    },
    {
      "id": 88,
      "name": "MON Sprint 33",
      "state": "closed"
    },
    {
      "id": 89,
      "name": "MON Sprint 34",
      "state": "closed"
    },
    {
      "id": 90,
      "name": "MON Sprint 35",
      "state": "closed"
    },
    {
      "id": 91,
      "name": "MON Sprint 36",
      "state": "closed"
    },
    {
      "id": 92,
      "name": "MON Sprint 37",
      "state": "closed"
    },
    {
      "id": 93,
      "name": "MON Sprint 38",
      "state": "closed"
    },
    {
      "id": 94,
      "name": "MON Sprint 39",
      "state": "closed"
    },
    {
      "id": 95,
      "name": "MON Sprint 40",
      "state": "closed"
    },
    {
      "id": 96,
      "name": "MON Sprint 41",
      "state": "closed"
    },
    {
      "id": 97,
      "name": "MON Sprint 42",
      "state": "closed"
    },
    {
      "id": 98,
      "name": "MON Sprint 43",
      "state": "closed"
    },
    {
      "id": 99,
      "name": "MON Sprint 44",
      "state": "closed"
    },
    {
      "id": 100,
      "name": "MON Sprint 45",
      "state": "closed"
    },
    {
      "id": 101,
      "name": "MON Sprint 46",
      "state": "closed"
    },
    {
      "id": 102,
      "name": "MON Sprint 47",
      "state": "closed"
    },
    {
      "id": 103,
      "name": "MON Sprint 48",
      "state": "closed"
    },
    {
      "id": 104,
      "name": "MON Sprint 50",
      "state": "closed"
    },
    {
      "id": 106,
      "name": "MON Sprint 51",
      "state": "closed"
    },
    {
      "id": 107,
      "name": "MON Sprint 52",
      "state": "closed"
    },
    {
      "id": 108,
      "name": "MON Sprint 53",
      "state": "closed"
    }
  ],
  "_meta": {
    "count": 50,
    "hasMore": true
  }
}
```

#### `-s active`

```
ID | NAME | STATE | START | END
125 | MON Sprint 70 | active | 2026-04-10 | 2026-04-24
```

#### `-s closed --max 3`

```
ID | NAME | STATE | START | END
58 | MON Sprint 1 | closed | 2023-08-14 | 2023-08-25
59 | MON Sprint 2 | closed | 2023-08-28 | 2023-09-08
60 | MON Sprint 3 | closed | 2023-09-11 | 2023-09-22
```

#### `-s future --max 3`

```
ID | NAME | STATE | START | END
126 | MON Sprint 71 | future | 2026-04-27 | 2026-05-08
127 | MON Sprint 72 | future | 2026-05-11 | 2026-05-22
128 | MON Sprint 73 | future | 2026-05-25 | 2026-06-05
```

#### Missing required `--board` flag

```
--board is required
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/agile/1.0/board/23/sprint"
# With state filter: ?state=active
```

### current

#### Default (table)

```
ID: 125
Name: MON Sprint 70
State: active
Start Date: 2026-04-10
End Date: 2026-04-24
```

#### `-o json`

```json
{
  "id": 125,
  "name": "MON Sprint 70",
  "state": "active"
}
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/agile/1.0/board/23/sprint?state=active"
```

### issues

#### Default (table)

```
KEY | SUMMARY | STATUS | ASSIGNEE | TYPE
MON-1918 | Q2 and Banno Notifications for MSR send | In Development | Devin Quirk | SDLC
MON-2846 | Add Funnel Chart & Interactive Stage Behavior t... | Backlog | Devin Quirk | SDLC
MON-3151 | Add additional identifiers to insights and acti... | Ready for Development | Devin Quirk | SDLC
```

#### `-o json`

```json
{
  "results": [
    {
      "key": "MON-1918",
      "summary": "Q2 and Banno Notifications for MSR send",
      "status": "In Development",
      "type": "SDLC",
      "assignee": "Devin Quirk"
    },
    {
      "key": "MON-2846",
      "summary": "Add Funnel Chart \u0026 Interactive Stage Behavior to Clients Page",
      "status": "Backlog",
      "type": "SDLC",
      "assignee": "Devin Quirk"
    },
    {
      "key": "MON-3151",
      "summary": "Add additional identifiers to insights and actions and audience builder exports",
      "status": "Ready for Development",
      "type": "SDLC",
      "assignee": "Devin Quirk"
    }
  ],
  "_meta": {
    "count": 3,
    "hasMore": true
  }
}
```

#### 404 error

```
fetching sprint issues: resource not found: We could not find the sprint
```

#### Equivalent API call

```bash
# Note: Agile endpoint is slow (~30s)
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/agile/1.0/sprint/125/issue?maxResults=3"
```
## links

Manage issue links.

### types

#### Default (table)

```
ID | NAME | OUTWARD | INWARD
10000 | Blocker | blocks | is blocked by
10001 | Cloners | clones | is cloned by
10002 | Duplicate | duplicates | is duplicated by
10006 | Polaris work item link | implements | is implemented by
10003 | Relates | relates to | relates to
```

#### `-o json`

```json
[
  {
    "id": "10000",
    "name": "Blocker",
    "inward": "is blocked by",
    "outward": "blocks"
  },
  {
    "id": "10001",
    "name": "Cloners",
    "inward": "is cloned by",
    "outward": "clones"
  },
  {
    "id": "10002",
    "name": "Duplicate",
    "inward": "is duplicated by",
    "outward": "duplicates"
  },
  {
    "id": "10006",
    "name": "Polaris work item link",
    "inward": "is implemented by",
    "outward": "implements"
  },
  {
    "id": "10003",
    "name": "Relates",
    "inward": "relates to",
    "outward": "relates to"
  }
]
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/issueLinkType"
```

### list

#### Default (table)

```
No links on MON-4810
```

#### `-o json` (empty state — prints text, not JSON)

```
# empty state: prints plain text even under -o json
No links on MON-4810
```

#### 404 error

```
resource not found: Issue does not exist or you do not have permission to see it.
```

#### Equivalent API call

```bash
# Links are embedded in the issue response
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/issue/MON-4810?fields=issuelinks"
```
## dashboards

Manage dashboards. (Requires Basic Auth — scoped tokens lack Dashboard scope.)

### list

#### Default (table)

```
ID | NAME | OWNER | FAVOURITE
10000 | Default dashboard |  | no
10001 | Epics | Rian Stockbower | yes
```

#### `-o json`

```json
[
  {
    "id": "10000",
    "name": "Default dashboard",
    "view": "/jira/dashboards/10000",
    "sharePermissions": [
      {
        "type": "global"
      }
    ]
  },
  {
    "id": "10001",
    "name": "Epics",
    "owner": {
      "accountId": "60e09bae7fcd820073089249",
      "displayName": "Rian Stockbower",
      "active": true,
      "avatarUrls": {
        "16x16": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/60e09bae7fcd820073089249/724b7268-43a2-4a44-a00e-7a204fe99f90/16",
        "24x24": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/60e09bae7fcd820073089249/724b7268-43a2-4a44-a00e-7a204fe99f90/24",
        "32x32": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/60e09bae7fcd820073089249/724b7268-43a2-4a44-a00e-7a204fe99f90/32",
        "48x48": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/60e09bae7fcd820073089249/724b7268-43a2-4a44-a00e-7a204fe99f90/48"
      }
    },
    "view": "/jira/dashboards/10001",
    "isFavourite": true,
    "popularity": 1
  }
]
```

#### `--search "Epics"`

```
ID | NAME | OWNER | FAVOURITE
10001 | Epics |  | no
```

#### No results

```
No dashboards found
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/dashboard?maxResults=5"
# With search: ?dashboardName=Epics
```

### get

Note: Makes two API calls — one for dashboard metadata, one for gadgets.

#### Default (table)

```
ID: 10001
Name: Epics
Owner: Rian Stockbower
URL: /jira/dashboards/10001
```

#### `-o json`

```json
{
  "dashboard": {
    "id": "10001",
    "name": "Epics",
    "owner": {
      "accountId": "60e09bae7fcd820073089249",
      "displayName": "Rian Stockbower",
      "active": true,
      "avatarUrls": {
        "16x16": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/60e09bae7fcd820073089249/724b7268-43a2-4a44-a00e-7a204fe99f90/16",
        "24x24": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/60e09bae7fcd820073089249/724b7268-43a2-4a44-a00e-7a204fe99f90/24",
        "32x32": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/60e09bae7fcd820073089249/724b7268-43a2-4a44-a00e-7a204fe99f90/32",
        "48x48": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/60e09bae7fcd820073089249/724b7268-43a2-4a44-a00e-7a204fe99f90/48"
      }
    },
    "view": "/jira/dashboards/10001",
    "isFavourite": true,
    "popularity": 1
  },
  "gadgets": []
}
```

#### 404 error

```
resource not found: The dashboard with id '99999' does not exist.
```

#### Equivalent API calls

```bash
# Call 1: dashboard metadata
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/dashboard/10001"
# Call 2: gadgets
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/dashboard/10001/gadget"
```

### gadgets list

#### Default (table)

```
No gadgets on dashboard 10001
```

#### `-o json` (empty state — prints text, not JSON)

```
# empty state: prints plain text even under -o json
No gadgets on dashboard 10001
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/dashboard/10001/gadget"
```

### gadgets remove

Removes a gadget from a dashboard by gadget ID (must be numeric).

#### Default (table)

```
Removed gadget 10121 from dashboard 10072
```

#### `-o json`

```json
{
  "dashboardId": "10072",
  "gadgetId": 10122,
  "status": "removed"
}
```

#### Error: gadget not found

```
resource not found: The dashboard gadget was not found.
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN -X DELETE \
  "https://monitproduct.atlassian.net/rest/api/3/dashboard/<dashboardId>/gadget/<gadgetId>"
```
## automation

Manage Jira automation rules. (Requires Basic Auth — scoped tokens lack Automation scope.)

### list

#### Default (table)

```
UUID | NAME | STATE
018c2840-57c1-7869-9393-11205cc87ce4 | ON/MON: Create Onboarding Tasks | ENABLED
```

#### `-o json`

```json
[
  {
    "uuid": "018c2840-57c1-7869-9393-11205cc87ce4",
    "name": "ON/MON: Create Onboarding Tasks",
    "state": "ENABLED",
    "description": "Creates Tasks when a new Onboarding Epic is created",
    "authorAccountId": "61292e4c4f29230069621c5f",
    "actorAccountId": "557058:f58131cb-b67d-43c7-b30d-6b58d40bd077",
    "ruleScopeARIs": [
      "ari:cloud:jira:217b4168-e429-4c0e-a2cd-263f3b695e73:project/10023",
      "ari:cloud:jira:217b4168-e429-4c0e-a2cd-263f3b695e73:project/10022"
    ]
  }
]
```

#### `--state ENABLED`

```
UUID | NAME | STATE
018c2840-57c1-7869-9393-11205cc87ce4 | ON/MON: Create Onboarding Tasks | ENABLED
```

#### `--state DISABLED`

```
No automation rules found
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/automation/rule/search" \
  -X POST -H "Content-Type: application/json" \
  -d '{"offset":0,"limit":50}'
```

### get

#### Default (table)

```
Name: ON/MON: Create Onboarding Tasks
UUID: 018c2840-57c1-7869-9393-11205cc87ce4
State: ENABLED
Description: Creates Tasks when a new Onboarding Epic is created
Components: 27 total — 4 condition(s), 23 action(s)
```

#### `--show-components`

```
Name: ON/MON: Create Onboarding Tasks
UUID: 018c2840-57c1-7869-9393-11205cc87ce4
State: ENABLED
Description: Creates Tasks when a new Onboarding Epic is created
Components: 27 total — 4 condition(s), 23 action(s)
# | COMPONENT | TYPE
1 | CONDITION | jira.jql.condition
2 | ACTION | jira.create.variable
3 | ACTION | jira.create.variable
4 | ACTION | jira.create.variable
5 | ACTION | jira.create.variable
6 | ACTION | jira.create.variable
7 | ACTION | jira.create.variable
8 | ACTION | jira.create.variable
9 | ACTION | jira.create.mapping-variable
10 | ACTION | jira.create.variable
11 | ACTION | jira.issue.create
12 | ACTION | jira.create.variable
13 | ACTION | jira.issue.create
14 | ACTION | jira.create.variable
15 | CONDITION | jira.condition.container.block
16 | CONDITION | jira.comparator.condition
17 | CONDITION | jira.comparator.condition
18 | ACTION | jira.issue.create
19 | ACTION | jira.issue.create
20 | ACTION | jira.issue.create
21 | ACTION | jira.issue.create
22 | ACTION | jira.issue.create
23 | ACTION | jira.issue.create
24 | ACTION | jira.issue.create
25 | ACTION | jira.issue.create
26 | ACTION | jira.issue.create
27 | ACTION | jira.issue.create
```

#### `-o json`

```json
{
  "id": "018c2840-57c1-7869-9393-11205cc87ce4",
  "name": "ON/MON: Create Onboarding Tasks",
  "state": "ENABLED",
  "componentSummary": "27 total — 4 condition(s), 23 action(s)"
}
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/automation/rule/018c2840-57c1-7869-9393-11205cc87ce4"
```

### export

Note: `-o` flag is ignored — output is always JSON regardless of format flag.

#### Default (pretty JSON)

```json
{
  "rule": {
    "name": "ON/MON: Create Onboarding Tasks",
    "state": "ENABLED",
    "description": "Creates Tasks when a new Onboarding Epic is created",
    "canOtherRuleTrigger": false,
    "notifyOnError": "FIRSTERROR",
    "authorAccountId": "61292e4c4f29230069621c5f",
    "actor": {
      "type": "ACCOUNT_ID",
      "actor": "557058:f58131cb-b67d-43c7-b30d-6b58d40bd077"
    },
    "trigger": {
      "component": "TRIGGER",
      "schemaVersion": 1,
      "type": "jira.issue.event.trigger:created",
      "value": {
        "eventKey": "jira:issue_created",
        "issueEvent": "issue_created",
        "eventFilters": [
          "ari:cloud:jira:217b4168-e429-4c0e-a2cd-263f3b695e73:project/10023",
          "ari:cloud:jira:217b4168-e429-4c0e-a2cd-263f3b695e73:project/10022"
        ]
      },
      "connectionId": null,
      "conditions": [],
      "id": "7354971"
    },
    "components": [
      {
...
```

#### `--compact` (minified JSON)

```json
{"rule":{"name":"ON/MON: Create Onboarding Tasks","state":"ENABLED","description":"Creates Tasks when a new Onboarding Epic is created","canOtherRuleTrigger":false,"notifyOnError":"FIRSTERROR","authorAccountId":"61292e4c4f29230069621c5f","actor":{"type":"ACCOUNT_ID","actor":"557058:f58131cb-b67d-43c7-b30d-6b58d40bd077"},"trigger":{"component":"TRIGGER","schemaVersion":1,"type":"jira.issue.event.trigger:created","value":{"eventKey":"jira:issue_created","issueEvent":"issue_created","eventFilters":["ari:cloud:jira:217b4168-e429-4c0e-a2cd-263f3b695e73:project/10023","ari:cloud:jira:217b4168-e429-4c0e-a2cd-263f3b695e73:project/10022"]},"connectionId":null,"conditions":[],"id":"7354971"},"components":[{"component":"CONDITION","schemaVersion":1,"type":"jira.jql.condition","value":"project = 'ON' AND issuetype = Epic","connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"7354972"},{"component":"ACTION","schemaVersion":1,"type":"jira.create.variable","value":{"id":"_customsmartvalue_id_1695939052449","name":{"type":"FREE","value":"bankName"},"type":"SMART","query":{"type":"SMART","value":"{{triggerIssue.customField_10036}}"},"lazy":false},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"7354973"},{"component":"ACTION","schemaVersion":1,"type":"jira.create.variable","value":{"id":"_customsmartvalue_id_1695939095835","name":{"type":"FREE","value":"bankPlatform"},"type":"SMART","query":{"type":"SMART","value":"{{triggerIssue.customField_10037}}"},"lazy":false},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"7354974"},{"component":"ACTION","schemaVersion":1,"type":"jira.create.variable","value":{"id":"_customsmartvalue_id_tenantId","name":{"type":"FREE","value":"tenantId"},"type":"SMART","query":{"type":"SMART","value":"{{triggerIssue.customField_10121}}"},"lazy":false},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"f785e5ea-384e-4c81-82e3-d10288c022f4"},{"component":"ACTION","schemaVersion":1,"type":"jira.create.variable","value":{"id":"_customsmartvalue_id_1695943399306","name":{"type":"FREE","value":"olbURL"},"type":"SMART","query":{"type":"SMART","value":"{{triggerIssue.customField_10050}}"},"lazy":false},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"7354976"},{"component":"ACTION","schemaVersion":1,"type":"jira.create.variable","value":{"id":"_customsmartvalue_id_1695939124525","name":{"type":"FREE","value":"dueDate"},"type":"SMART","query":{"type":"SMART","value":"{{triggerIssue.duedate}}"},"lazy":false},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"7354977"},{"component":"ACTION","schemaVersion":1,"type":"jira.create.variable","value":{"id":"_customsmartvalue_id_products","name":{"type":"FREE","value":"products"},"type":"SMART","query":{"type":"SMART","value":"{{triggerIssue.customField_10154}}"},"lazy":false},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"4657877f-a212-4507-959d-bb0fe8b87768"},{"component":"ACTION","schemaVersion":1,"type":"jira.create.variable","value":{"id":"_customsmartvalue_id_pplambdaurl","name":{"type":"FREE","value":"ppLambdaUrl"},"type":"SMART","query":{"type":"SMART","value":"{{triggerIssue.customField_10155}}"},"lazy":false},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"fc5705fa-8987-4c3c-a5f0-6e8c99fb2d80"},{"component":"ACTION","schemaVersion":1,"type":"jira.create.mapping-variable","value":{"name":{"type":"FREE","value":"fiAddendum"},"mappings":[{"key":"Apiture","value":"The Apiture ID is *<institution id>*"}]},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"7354980"},{"component":"ACTION","schemaVersion":1,"type":"jira.create.variable","value":{"id":"_customsmartvalue_id_productDisplayName","name":{"type":"FREE","value":"productDisplayName"},"type":"SMART","query":{"type":"SMART","value":"{{triggerIssue.customField_10189}}"},"lazy":false},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"c4799861-b26f-4b63-a1aa-b11f52837522"},{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"Provision FI for {{bankName}} / {{tenantId}} ({{bankPlatform}})"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"FI Identifier should be *{{tenantId}}*\n{{fiAddendum.get(bankPlatform)}}\n\n"},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10023"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10007"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"FI Provisioning"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}},{"field":{"type":"ID","value":"customfield_10035"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:float","type":"SET","value":"1"},{"field":{"type":"ID","value":"customfield_10005"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"id":"10034","value":"Feature"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"7354982"},{"component":"ACTION","schemaVersion":1,"type":"jira.create.variable","value":{"id":"_customsmartvalue_id_1695919100340","name":{"type":"FREE","value":"fiProvisionIssueKey"},"type":"SMART","query":{"type":"SMART","value":"{{createdIssue.key}}"},"lazy":false},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"7354983"},{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"Initialize FI data storage for {{bankName}} / {{tenantId}} ({{bankPlatform}})"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"Add the FI to the PRD *segregated_data_fi_ids* list in Terraform."},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10022"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10025"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"FI Onboarding"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}},{"field":{"type":"ID","value":"customfield_10035"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:float","type":"SET","value":"1"},{"field":{"type":"ID","value":"customfield_10005"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"id":"10034","value":"Feature"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"7354984"},{"component":"ACTION","schemaVersion":1,"type":"jira.create.variable","value":{"id":"_customsmartvalue_id_1695919100340","name":{"type":"FREE","value":"fiDataStorageKey"},"type":"SMART","query":{"type":"SMART","value":"{{createdIssue.key}}"},"lazy":false},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"7354985"},{"component":"CONDITION","schemaVersion":1,"type":"jira.condition.container.block","value":{},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[{"component":"CONDITION_BLOCK","schemaVersion":1,"type":"jira.condition.if.block","value":{"conditionMatchType":"ALL"},"connectionId":null,"conditions":[{"component":"CONDITION","schemaVersion":1,"type":"jira.comparator.condition","value":{"first":"{{bankPlatform}}","second":"Banno","operator":"EQUALS"},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":"7354987","children":[],"id":"7354988"}],"parentId":"7354986","conditionParentId":null,"children":[{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"Banno Platform Plugin Setup for {{bankName}} / {{tenantId}} ({{bankPlatform}})"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"Work with Bank's \"Banno Admin Team\" to implement our Plugin in their system - [details|https://monitproduct.atlassian.net/wiki/spaces/PLAYBOOK/pages/2444394497/Onboarding+a+Banno+Platform+Bank#Plugin-Configuration-(Bank-Side)]\n\nWe will need to provide:\n* Standard settings plus the Redirect URLs specific to the Bank's FI for the \"External Application\"\n* Standard settings for the \"Plugin Card\"\n** If the Bank has desires an alternative Product Display Name ({{productDisplayName}}), they should enter that as well (to align with the full app experience).\n\nWe will need to collect the following information (for subsequent Monit side configuration) from the Bank:\n* API Domain\n* Client ID\n* Client Secret\n\nThe API Domain should just be the domain of their online banking website. The Client Id and Client Secret will be generated as part of the Plugin setup; ask the Bank team to transmit them to Monit via a secure method."},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10023"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10007"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"Platform Config"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}},{"field":{"type":"ID","value":"customfield_10035"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:float","type":"SET","value":"1"},{"field":{"type":"ID","value":"customfield_10005"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"id":"10034","value":"Feature"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":"7354987","conditionParentId":null,"children":[],"id":"7354989"},{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"Monit App Configuration Update for Bank Onboarding for {{bankName}} / {{tenantId}} ({{bankPlatform}})"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"Update the Monit App configuration in Terraform\n* Create a new Tenant Secret resource for the bank\n* Add the Tenant Secret to the Banno Auth Lambda's access policies\n* Allow the bank's domain to iFrame Monit App\n** Online Banking URL: {{olbURL}}"},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10022"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10025"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"FI Onboarding"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}},{"field":{"type":"ID","value":"customfield_10035"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:float","type":"SET","value":"2"},{"field":{"type":"ID","value":"customfield_10005"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"id":"10034","value":"Feature"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":"7354987","conditionParentId":null,"children":[],"id":"7354990"},{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"Initialize Tenant Credentials for {{bankName}} / {{tenantId}} ({{bankPlatform}})"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"Update AWS Secrets\n* Tenant Secret\n** apiDomain/bannoApiDomain\n** apiClientId/bannoClientId\n** apiClientSecre/bannoClientSecret\n* *sso-monitsso/banno-identity-provider* Secret\n** Add Client ID and Client Secret to JSON structure"},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10023"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10007"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"FI Credentials"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}},{"field":{"type":"ID","value":"customfield_10035"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:float","type":"SET","value":"1"},{"field":{"type":"ID","value":"customfield_10005"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"id":"10034","value":"Feature"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":"7354987","conditionParentId":null,"children":[],"id":"7354991"},{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"Monit SSO Configuration Update for Bank Onboarding for {{bankName}} / {{tenantId}} ({{bankPlatform}})"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"Update the Monit SSO configuration in Terraform\n* Create an IDP for the bank\n* Allow the bank's domain to iFrame Monit SSO\n** Online Banking URL: {{olbURL}}"},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10022"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10025"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"FI Onboarding"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}},{"field":{"type":"ID","value":"customfield_10035"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:float","type":"SET","value":"2"},{"field":{"type":"ID","value":"customfield_10005"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"id":"10034","value":"Feature"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":"7354987","conditionParentId":null,"children":[],"id":"7354992"}],"id":"7354987"},{"component":"CONDITION_BLOCK","schemaVersion":1,"type":"jira.condition.if.block","value":{"conditionMatchType":"ALL"},"connectionId":null,"conditions":[{"component":"CONDITION","schemaVersion":1,"type":"jira.comparator.condition","value":{"first":"{{bankPlatform}}","second":"Narmi","operator":"EQUALS"},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":"7354993","children":[],"id":"7354994"}],"parentId":"7354986","conditionParentId":null,"children":[{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"Narmi Integration Setup for {{bankName}} / {{tenantId}} ({{bankPlatform}})"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"Work with Narmi Admin Team to setup our integration in the bank's system - [details|https://monitproduct.atlassian.net/wiki/spaces/PLAYBOOK/pages/2480144390/Onboarding+a+Narmi+Platform+Bank#iframe-Configuration-(Bank-Side)]\n\nWe will need to provide:\n* Redirect URL\n* Product Display Name: {{productDisplayName}} (default: \"Business Insights\")\n* Bank's Desired Icon\n\nWe will need to collect the following information (for subsequent Monit side configuration) from the Bank:\n* API Domain\n* Client ID\n* Client Secret\n\nThe API Domain should just be the domain of their online banking website. The Client Id and Client Secret will be generated as part of the Plugin setup; ask the Narmi team to transmit them to Monit via a secure method."},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10023"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10007"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"Platform Config"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}},{"field":{"type":"ID","value":"customfield_10035"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:float","type":"SET","value":"1"},{"field":{"type":"ID","value":"customfield_10005"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"id":"10034","value":"Feature"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":"7354993","conditionParentId":null,"children":[],"id":"7354995"},{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"Monit App Configuration Update for Bank Onboarding for {{bankName}} / {{tenantId}} ({{bankPlatform}})"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"Update the Monit App configuration in Terraform\n* Create a new Tenant Secret resource for the bank\n* Add the Tenant Secret to the Narmi Auth Lambda's access policies\n* Allow the bank's domain to iFrame Monit App\n** Online Banking URL: {{olbURL}}"},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10022"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10025"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"FI Onboarding"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}},{"field":{"type":"ID","value":"customfield_10035"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:float","type":"SET","value":"2"},{"field":{"type":"ID","value":"customfield_10005"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"id":"10034","value":"Feature"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":"7354993","conditionParentId":null,"children":[],"id":"7354996"},{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"Initialize Tenant Credentials for {{bankName}} / {{tenantId}} ({{bankPlatform}})"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"Update AWS Secrets\n* Tenant Secret\n** apiDomain\n** apiClientId\n** apiClientSecret\n"},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10023"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10007"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"FI Credentials"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}},{"field":{"type":"ID","value":"customfield_10035"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:float","type":"SET","value":"1"},{"field":{"type":"ID","value":"customfield_10005"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"id":"10034","value":"Feature"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":"7354993","conditionParentId":null,"children":[],"id":"7354997"},{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"Monit SSO Configuration Update for Bank Onboarding for {{bankName}} / {{tenantId}} ({{bankPlatform}})"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"Update the Monit SSO configuration in Terraform\n* Create an IDP for the bank\n* Allow the bank's domain to iFrame Monit SSO\n** Online Banking URL: {{olbURL}}"},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10022"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10025"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"FI Onboarding"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}},{"field":{"type":"ID","value":"customfield_10035"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:float","type":"SET","value":"2"},{"field":{"type":"ID","value":"customfield_10005"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"id":"10034","value":"Feature"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":"7354993","conditionParentId":null,"children":[],"id":"7354998"}],"id":"7354993"},{"component":"CONDITION_BLOCK","schemaVersion":1,"type":"jira.condition.if.block","value":{"conditionMatchType":"ALL"},"connectionId":null,"conditions":[{"component":"CONDITION","schemaVersion":1,"type":"jira.comparator.condition","value":{"first":"{{bankPlatform}}","second":"Q2","operator":"EQUALS"},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":"7354999","children":[],"id":"7355000"}],"parentId":"7354986","conditionParentId":null,"children":[{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"Monit App Configuration Update for Bank Onboarding for {{bankName}} / {{tenantId}} ({{bankPlatform}})"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"Update the Monit App configuration in Terraform\n* Allow the bank's domain to iFrame Monit App\n** Online Banking URL: {{olbURL}}"},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10022"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10025"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"FI Onboarding"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}},{"field":{"type":"ID","value":"customfield_10035"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:float","type":"SET","value":"2"},{"field":{"type":"ID","value":"customfield_10005"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"id":"10034","value":"Feature"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":"7354999","conditionParentId":null,"children":[],"id":"7355001"},{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"Monit SSO Configuration Update for Bank Onboarding for {{bankName}} / {{tenantId}} ({{bankPlatform}})"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"Update the Monit SSO configuration in Terraform\n* Create an IDP for the bank\n* Allow the bank's domain to iFrame Monit SSO\n** Online Banking URL: {{olbURL}}"},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10022"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10025"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"FI Onboarding"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}},{"field":{"type":"ID","value":"customfield_10035"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:float","type":"SET","value":"2"},{"field":{"type":"ID","value":"customfield_10005"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"id":"10034","value":"Feature"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":"7354999","conditionParentId":null,"children":[],"id":"7355002"},{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"Q2 Extension Installation for {{bankName}} / {{tenantId}} ({{bankPlatform}})"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"Work with Q2 Team to install our extension into the bank's system - [details|https://monitproduct.atlassian.net/wiki/spaces/PLAYBOOK/pages/2500067362/Onboarding+a+Q2+Platform+Bank#Configuration-(Q2-Side)]\n\nWe manage part of the installation setup through Q2's Developer Portal Self-Service. Specifically, we can specify the Bank-specific \"Vendor Config\", which defines the configuration details for the extension.\n\nAdditional steps must be completed by Q2, request via a [Support Ticket|https://www.q2developer.com/support/create].\n* Our SAML Certificate must be copied from our Vault to the FI's App\n* The Extension must be configured specifically to open in the \"Overpanel\"\n"},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10023"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10007"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"Platform Config"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}},{"field":{"type":"ID","value":"customfield_10035"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:float","type":"SET","value":"2"},{"field":{"type":"ID","value":"customfield_10005"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"id":"10034","value":"Feature"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":"7354999","conditionParentId":null,"children":[],"id":"7355003"}],"id":"7354999"},{"component":"CONDITION_BLOCK","schemaVersion":1,"type":"jira.condition.if.block","value":{"conditionMatchType":"ALL"},"connectionId":null,"conditions":[{"component":"CONDITION","schemaVersion":1,"type":"jira.comparator.condition","value":{"first":"{{bankPlatform}}","second":"Apiture","operator":"EQUALS"},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":"7355004","children":[],"id":"7355005"}],"parentId":"7354986","conditionParentId":null,"children":[{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"Configure tenant settings for {{bankName}} / {{tenantId}} ({{bankPlatform}})"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"By default, no tenant settings will exist for the FI. The process of [reconciliation|https://monitproduct.atlassian.net/wiki/spaces/ENG/pages/2527854596/Managing+Tenant+Settings#Step-3---Propagating-the-setting-to-existing-FIs] will create settings for new FIs.\n* Default settings will be applied, then the settings below can be applied.\n\n*Products:* {{products}}\n{{#if(ppLambdaUrl)}}*PositivePay Lambda URL:* {{ppLambdaUrl}}\n{{/}}\n\nSpecified Settings\n* \"display-bank-info\" should be OFF\n* invite-codes should be OFF\n* display-chat should be OFF\n* display-sage-50-uk should be OFF\n* display-qbd should be OFF\n* display-sandbox-integrations should be OFF\n* display-multiple-businesses should be OFF <-- {{bankPlatform}} platform banks should not ever have this enabled, due to how the {{bankPlatform}} UI is organized\n\nEverything else should be ON."},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10023"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10007"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"Tenant Settings"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}},{"field":{"type":"ID","value":"customfield_10035"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:float","type":"SET","value":"1"},{"field":{"type":"ID","value":"customfield_10005"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"id":"10034","value":"Feature"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":"7355004","conditionParentId":null,"children":[],"id":"7355006"}],"id":"7355004"},{"component":"CONDITION_BLOCK","schemaVersion":1,"type":"jira.condition.if.block","value":{"conditionMatchType":"ALL"},"connectionId":null,"conditions":[],"parentId":"7354986","conditionParentId":null,"children":[{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"Configure tenant settings for {{bankName}} / {{tenantId}} ({{bankPlatform}})"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"By default, no tenant settings will exist for the FI. The process of [reconciliation|https://monitproduct.atlassian.net/wiki/spaces/ENG/pages/2527854596/Managing+Tenant+Settings#Step-3---Propagating-the-setting-to-existing-FIs] will create settings for new FIs.\n* Default settings will be applied, then the settings below can be applied.\n\n*Products:* {{products}}\n{{#if(ppLambdaUrl)}}*PositivePay Lambda URL:* {{ppLambdaUrl}}\n{{/}}\n\nSpecified Settings\n* \"display-bank-info\" should be OFF\n* invite-codes should be OFF\n* display-chat should be OFF\n* display-sage-50-uk should be OFF\n* display-qbd should be OFF\n* display-sandbox-integrations should be OFF\n\nEverything else should be ON."},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10023"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10007"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"Tenant Settings"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}},{"field":{"type":"ID","value":"customfield_10035"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:float","type":"SET","value":"1"},{"field":{"type":"ID","value":"customfield_10005"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"id":"10034","value":"Feature"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":"7355007","conditionParentId":null,"children":[],"id":"7355008"}],"id":"7355007"}],"id":"7354986"},{"component":"CONDITION","schemaVersion":1,"type":"jira.comparator.condition","value":{"first":"{{bankPlatform}}","second":"Q2","operator":"EQUALS"},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"50f7f01f-9ddc-46e9-a078-479a0c127eab"},{"component":"CONDITION","schemaVersion":1,"type":"jira.comparator.condition","value":{"first":"{{products}}","second":"CheckSync","operator":"CONTAINS"},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"02906d2e-d4e6-40ce-a05b-632cdf44cc3c"},{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"Provision CheckSync OAuth for {{bankName}} / {{tenantId}} (Q2)"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"Provision Q2 CheckSync OAuth credentials and deploy secrets for {{bankName}} / {{tenantId}}.\n\nPrerequisites (Q2-side must complete first):\n* Email Q2 requesting ETMS API access\n* Q2 creates Caliper API Application and provides ETMS URLs\n* FI completes Centrix/ETMS prerequisite questionnaire\n\nSteps:\n# Gather ETMS details from Q2 (domain URL and relative API path)\n# Run Playwright automation to create OAuth app in Q2 Developer Portal\n#* Creates app: Monit-{{bankName}}-Production-Centrix Exacttms Connect\n#* Selects ExactTMS-Connect-API as resource\n#* Extracts client_id and client_secret\n# Update checksync CSV with credentials\n# Deploy secrets to AWS Secrets Manager (prd-monitapp/q2-api-fi-secrets/{{tenantId}})\n# Enable tenant settings: has-checksync = true, uses-positive-pay = true\n# Verify secret exists in AWS\n\nTooling:\n* ~/monit/devops-utils/checksync-secrets/provision_q2_oauth.py (Playwright)\n* ~/monit/devops-utils/checksync-secrets/create_checksync_secrets.py (AWS secrets)\n* q2-checksync-provisioning Claude skill"},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10023"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10007"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"Platform Config"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}},{"field":{"type":"ID","value":"customfield_10035"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:float","type":"SET","value":"2"},{"field":{"type":"ID","value":"customfield_10005"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"id":"10034","value":"Feature"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"1f027cc4-089e-49e7-928d-69ef5e323c05"},{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"Create Customer Success Email for {{bankName}} / {{tenantId}} ({{bankPlatform}})"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"Should have Ryan and Rian on it to start.\n\n"},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10023"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10007"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"CS Email"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}},{"field":{"type":"ID","value":"customfield_10035"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:float","type":"SET","value":"1"},{"field":{"type":"ID","value":"customfield_10005"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"id":"10034","value":"Feature"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"7355009"},{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"Provision CS Admin Access for {{bankName}} / {{tenantId}} ({{bankPlatform}})"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"Setup Bank Portal access for the CS Email *{{tenantId}}@monitapp.io*\n* [Banker User Onboarding|https://monitproduct.atlassian.net/wiki/spaces/PLAYBOOK/pages/2532016129/Banker+User+Onboarding]\n\nAdd credentials to *Customer Success Airlock* vault in 1Password\n\n*<Add any specific access adjustments as needed>*\n"},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10023"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10007"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"CS Admin Access"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}},{"field":{"type":"ID","value":"customfield_10035"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:float","type":"SET","value":"1"},{"field":{"type":"ID","value":"customfield_10005"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"id":"10034","value":"Feature"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"7355010"},{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"Create Theming & Branding for {{bankName}} / {{tenantId}} ({{bankPlatform}})"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"Details to confirm:\n* Custom Fonts to be used\n* Product Display Name: {{productDisplayName}} (default: \"Business Insights\" if blank)\n* Application Support Information\n* Application Footer Details"},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10023"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10007"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"Theme Creation"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}},{"field":{"type":"ID","value":"customfield_10035"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:float","type":"SET","value":"1"},{"field":{"type":"ID","value":"customfield_10005"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"id":"10034","value":"Feature"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"7355011"},{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"Implement Bank Theme for {{bankName}} / {{tenantId}} ({{bankPlatform}})"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"Migrate bank theme to PRD under FI Identifier\nCoordinate with Ryan Johnson, to schedule copying the theme over *monit-theme-demo* in PRD, to enable video and screenshot generation\n"},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10023"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10007"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"Theme Implementation"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}},{"field":{"type":"ID","value":"customfield_10035"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:float","type":"SET","value":"1"},{"field":{"type":"ID","value":"customfield_10005"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"id":"10034","value":"Feature"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"7355012"},{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"Add {{bankName}} / {{tenantId}} ({{bankPlatform}}) to billing system"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"\n"},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10023"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10007"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"Billing"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}},{"field":{"type":"ID","value":"assignee"},"fieldType":"assignee","type":"SET","value":{"type":"ID","value":"5d914e721d47a50c34d4689b"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"7355014"},{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"Add {{bankName}} / {{tenantId}} ({{bankPlatform}}) to QBO"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"\n"},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10023"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10007"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"Billing"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}},{"field":{"type":"ID","value":"assignee"},"fieldType":"assignee","type":"SET","value":{"type":"ID","value":"5d914e721d47a50c34d4689b"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"15fa70a2-8ffb-463e-a560-5c1b3ee0223d"},{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"Add {{bankName}} / {{tenantId}} to sales commission sheet"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"\n"},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10023"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10007"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"Billing"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}},{"field":{"type":"ID","value":"assignee"},"fieldType":"assignee","type":"SET","value":{"type":"ID","value":"5d914e721d47a50c34d4689b"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"1087c518-7124-4ccf-89dd-b2dc57b29979"},{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"Add {{bankName}} / {{tenantId}} to executive outreach schedule"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"\n"},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10023"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10007"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"Billing"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}},{"field":{"type":"ID","value":"assignee"},"fieldType":"assignee","type":"SET","value":{"type":"ID","value":"5d914e721d47a50c34d4689b"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"0f90a890-9417-468d-8008-3077044127b6"},{"component":"ACTION","schemaVersion":12,"type":"jira.issue.create","value":{"operations":[{"field":{"type":"ID","value":"summary"},"fieldType":"summary","type":"SET","value":"UAT Preflight Checklist for {{bankName}} / {{tenantId}} ({{bankPlatform}})"},{"field":{"type":"ID","value":"description"},"fieldType":"description","type":"SET","value":"Review and validate the [Preflight Checklist|https://monitproduct.atlassian.net/wiki/spaces/CUS/pages/2539978753/Preflight+checklist+for+new+FI+launch] for {{bankName}}\n"},{"field":{"type":"ID","value":"project"},"fieldType":"project","type":"SET","value":{"type":"ID","value":"10023"}},{"field":{"type":"ID","value":"issuetype"},"fieldType":"issuetype","type":"SET","value":{"type":"ID","value":"10007"}},{"field":{"type":"ID","value":"components"},"fieldType":"components","type":"SET","value":[{"type":"NAME","value":"Preflight UAT"}]},{"field":{"type":"NAME","value":"Bank Name"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:textfield","type":"SET","value":"{{bankName}}"},{"field":{"type":"NAME","value":"Banking Platform"},"fieldType":"com.atlassian.jira.plugin.system.customfieldtypes:select","type":"SET","value":{"type":"SMART","value":"{{bankPlatform}}"}},{"field":{"type":"ID","value":"duedate"},"fieldType":"duedate","type":"SET","value":"{{dueDate}}"},{"field":{"type":"ID","value":"parent"},"fieldType":"parent","type":"SET","value":{"type":"COPY","value":"trigger"}}],"advancedFields":null,"sendNotifications":false},"connectionId":null,"conditions":[],"parentId":null,"conditionParentId":null,"children":[],"id":"7355018"}],"ruleScopeARIs":["ari:cloud:jira:217b4168-e429-4c0e-a2cd-263f3b695e73:project/10023","ari:cloud:jira:217b4168-e429-4c0e-a2cd-263f3b695e73:project/10022"],"labels":[],"writeAccessType":"UNRESTRICTED","collaborators":[],"uuid":"018c2840-57c1-7869-9393-11205cc87ce4","created":1701482354.625000000,"updated":1773422582.508000000},"connections":[]}
...
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/automation/rule/018c2840-57c1-7869-9393-11205cc87ce4"
```
## fields

Manage Jira custom fields.

### list

#### Default (table, first 20 lines)

```
ID | NAME | TYPE | CUSTOM
statuscategorychangedate | Status Category Changed | datetime | no
fixVersions | Fix versions | array | no
statusCategory | Status Category | statusCategory | no
parent | Parent |  | no
resolution | Resolution | resolution | no
lastViewed | Last Viewed | datetime | no
priority | Priority | priority | no
customfield_10189 | Product Display Name | string | yes
labels | Labels | array | no
timeestimate | Remaining Estimate | number | no
aggregatetimeoriginalestimate | Σ Original Estimate | number | no
versions | Affects versions | array | no
issuelinks | Linked Issues | array | no
assignee | Assignee | user | no
status | Status | status | no
components | Components | array | no
issuekey | Key |  | no
customfield_10050 | Online Banking URL | string | yes
customfield_10051 | Design | array | yes
```

#### `--custom` (first 20 lines)

```
ID | NAME | TYPE | CUSTOM
customfield_10189 | Product Display Name | string | yes
customfield_10050 | Online Banking URL | string | yes
customfield_10051 | Design | array | yes
customfield_10052 | Vulnerability | any | yes
customfield_10053 | Sentiment | array | yes
customfield_10054 | Goals | array | yes
customfield_10055 | Focus Areas | array | yes
customfield_10049 | Migration Archive | string | yes
customfield_10040 | Send Gainsight Emails | option | yes
customfield_10041 | Send MSR Emails | option | yes
customfield_10043 | Category | option | yes
customfield_10044 | Meta Status | array | yes
customfield_10046 | QA Notes | string | yes
customfield_10039 | Theming & Branding Info | string | yes
customfield_10030 | Total forms | number | yes
customfield_10031 | Project overview key | string | yes
customfield_10032 | Project overview status | string | yes
customfield_10154 | Products | array | yes
customfield_10155 | PPLambdaUrl | string | yes
```

#### `--name "story"`

```
ID | NAME | TYPE | CUSTOM
customfield_10035 | Story Points | number | yes
customfield_10016 | Story point estimate | number | yes
```

#### `--name "story" -o json`

```json
[
  {
    "id": "customfield_10035",
    "key": "customfield_10035",
    "name": "Story Points",
    "custom": true,
    "orderable": true,
    "navigable": true,
    "searchable": true,
    "schema": {
      "type": "number",
      "custom": "com.atlassian.jira.plugin.system.customfieldtypes:float",
      "customId": 10035
    },
    "clauseNames": [
      "cf[10035]",
      "Story Points[Number]",
      "Story Points"
    ]
  },
  {
    "id": "customfield_10016",
    "key": "customfield_10016",
    "name": "Story point estimate",
    "custom": true,
    "orderable": true,
    "navigable": true,
    "searchable": true,
    "schema": {
      "type": "number",
      "custom": "com.pyxis.greenhopper.jira:jsw-story-points",
      "customId": 10016
    },
    "clauseNames": [
      "cf[10016]",
      "Story point estimate"
    ]
  }
]
```

#### `--custom --name "story"`

```
ID | NAME | TYPE | CUSTOM
customfield_10035 | Story Points | number | yes
customfield_10016 | Story point estimate | number | yes
```

#### No results

```
No fields found
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/field"
# Custom only: filter client-side on schema.custom presence
```

### contexts list

#### Default (table)

```
ID | NAME | GLOBAL | ANY_ISSUE_TYPE
10139 | Default Configuration Scheme for Banking Platform | yes | yes
10178 | Engineering | no | yes
```

#### `-o json`

```json
[
  {
    "id": "10139",
    "name": "Default Configuration Scheme for Banking Platform",
    "description": "Default configuration scheme generated by Jira",
    "isGlobalContext": true,
    "isAnyIssueType": true
  },
  {
    "id": "10178",
    "name": "Engineering",
    "isGlobalContext": false,
    "isAnyIssueType": true
  }
]
```

#### 404 error

```
fetching field contexts: resource not found: The custom field was not found.
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/field/customfield_10037/context"
```

### options list

#### Default (table)

```
ID | VALUE | DISABLED
10020 | Apiture | no
10021 | Banno | no
10022 | Narmi | no
10023 | Q2 | no
10030 | No Platform | no
```

#### `-o json`

```json
[
  {
    "id": "10020",
    "value": "Apiture",
    "disabled": false
  },
  {
    "id": "10021",
    "value": "Banno",
    "disabled": false
  },
  {
    "id": "10022",
    "value": "Narmi",
    "disabled": false
  },
  {
    "id": "10023",
    "value": "Q2",
    "disabled": false
  },
  {
    "id": "10030",
    "value": "No Platform",
    "disabled": false
  }
]
```

#### Equivalent API call

```bash
# Auto-detects default context; explicit: ?contextId=CONTEXT_ID
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/field/customfield_10037/context/option"
```
## comments

Manage issue comments.

### list

#### Default (table)

```
ID | AUTHOR | CREATED | BODY
21242 | Aaron Wong | 2026-04-16 | Short audit conclusion after the current code changes: The major source-level accessibility findings... [truncated, use --no-truncate for complete text]
```

#### `-o json`

```json
{
  "results": [
    {
      "id": "21242",
      "author": "Aaron Wong",
      "created": "2026-04-16",
      "body": "Short audit conclusion after the current code changes:\nThe major source-level accessibility findings on CapOne-specific surfaces appear to be addressed or materially improved:\n- loading / redirect sta..."
    }
  ],
  "_meta": {
    "count": 1,
    "hasMore": false
  }
}
```

#### `--no-truncate`

```
ID: 21242
Author: Aaron Wong
Created: 2026-04-16
Body: Short audit conclusion after the current code changes:
The major source-level accessibility findings on CapOne-specific surfaces appear to be addressed or materially improved:
- loading / redirect states now expose accessible status messaging
- the unsupported-package modal now exposes both title and description correctly
- step triggers now have stronger names/state semantics
- reviewed decorative imagery is no longer over-announced where visible text already carries the meaning

The main remaining risk is no longer an obvious code-level defect. It is runtime validation of the interactive step-preview surface and live-region behavior against real keyboard / assistive-technology behavior.
Practical readout:
- Resolved or materially improved at code level: loading status, modal description wiring, weak trigger labels, redundant decorative image announcements
- Still requires manual validation: CaponeStepsPreview interaction/focus behavior, loading announcement behavior across real SR/browser combinations, and rendered focus visibility

Bottom line: this now looks much stronger from a source audit perspective; remaining uncertainty is primarily manual conformance validation, not missing obvious ARIA wiring.
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/issue/MON-4810/comment?maxResults=50"
```
## transitions

Manage issue workflow transitions.

### list

#### Default (table)

```
ID | NAME | TO STATUS
121 | Ready to QA | Ready for QA
151 | Return to Development | In Development
11 | Backlog | Backlog
21 | Ready for Development | Ready for Development
31 | In Development | In Development
41 | In Code Review | In Code Review
51 | Ready for QA | Ready for QA
61 | Ready for Deployment | Ready for Deployment
71 | Deployed | Deployed
81 | Canceled | Canceled
```

#### `-o json`

```json
[
  {
    "id": "121",
    "name": "Ready to QA",
    "to": {
      "id": "10008",
      "name": "Ready for QA",
      "statusCategory": {
        "id": 4,
        "key": "indeterminate",
        "name": "In Progress"
      }
    }
  },
  {
    "id": "151",
    "name": "Return to Development",
    "to": {
      "id": "10005",
      "name": "In Development",
      "statusCategory": {
        "id": 4,
        "key": "indeterminate",
        "name": "In Progress"
      }
    }
  },
  {
    "id": "11",
    "name": "Backlog",
    "to": {
      "id": "10003",
      "name": "Backlog",
      "statusCategory": {
        "id": 2,
        "key": "new",
        "name": "To Do"
      }
    }
  },
  {
    "id": "21",
    "name": "Ready for Development",
    "to": {
      "id": "10004",
      "name": "Ready for Development",
      "statusCategory": {
        "id": 2,
        "key": "new",
        "name": "To Do"
      }
    }
  },
  {
    "id": "31",
    "name": "In Development",
    "to": {
      "id": "10005",
      "name": "In Development",
      "statusCategory": {
        "id": 4,
        "key": "indeterminate",
        "name": "In Progress"
      }
    }
  },
  {
    "id": "41",
    "name": "In Code Review",
    "to": {
      "id": "10006",
      "name": "In Code Review",
      "statusCategory": {
        "id": 4,
        "key": "indeterminate",
        "name": "In Progress"
      }
    }
  },
  {
    "id": "51",
    "name": "Ready for QA",
    "to": {
      "id": "10008",
      "name": "Ready for QA",
      "statusCategory": {
        "id": 4,
        "key": "indeterminate",
        "name": "In Progress"
      }
    }
  },
  {
    "id": "61",
    "name": "Ready for Deployment",
    "to": {
      "id": "10009",
      "name": "Ready for Deployment",
      "statusCategory": {
        "id": 4,
        "key": "indeterminate",
        "name": "In Progress"
      }
    }
  },
  {
    "id": "71",
    "name": "Deployed",
    "to": {
      "id": "10010",
      "name": "Deployed",
      "statusCategory": {
        "id": 3,
        "key": "done",
        "name": "Done"
      }
    }
  },
  {
    "id": "81",
    "name": "Canceled",
    "to": {
      "id": "10011",
      "name": "Canceled",
      "statusCategory": {
        "id": 3,
        "key": "done",
        "name": "Done"
      }
    }
  }
]
```

#### `--fields` (show required fields per transition)

```
ID | NAME | TO STATUS | REQUIRED FIELDS
121 | Ready to QA | Ready for QA | -
151 | Return to Development | In Development | -
11 | Backlog | Backlog | -
21 | Ready for Development | Ready for Development | -
31 | In Development | In Development | -
41 | In Code Review | In Code Review | -
51 | Ready for QA | Ready for QA | -
61 | Ready for Deployment | Ready for Deployment | -
71 | Deployed | Deployed | -
81 | Canceled | Canceled | -
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/issue/MON-4810/transitions"
```
## attachments

Manage issue attachments.

### list

#### Default (table)

```
No attachments found on MON-4810
```

#### `-o json` (empty state — prints text, not JSON)

```
# empty state: prints plain text even under -o json
No attachments found on MON-4810
```

#### Equivalent API call

```bash
# Attachments are embedded in the issue response
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/issue/MON-4810?fields=attachment"
```

### add

Uploads one or more files as attachments to an issue. The attachment ID is embedded in the success output.

**Note:** No `-o json` branch — output is always the presenter text.

#### Default (table)

```
Uploaded jtk-test-attachment.txt (ID: 15974, Size: 49 B)
```

#### Error: file not found

```
file not found: /tmp/does-not-exist.txt
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN -X POST \
  -H "X-Atlassian-Token: no-check" \
  -F "file=@/path/to/file.txt" \
  "https://monitproduct.atlassian.net/rest/api/3/issue/MON-4810/attachments"
```

### get (download)

Downloads an attachment by ID. `-o` is a **download path** (directory or filename), not an output format selector.

**Note:** No `-o json` branch — output is always the presenter text.

#### Default (table)

```
Downloaded /tmp/jtk-test-attachment.txt (49 B)
```

#### Error: attachment not found

```
getting attachment: fetching attachment: resource not found: The attachment with id '99999999' does not exist
```

#### Equivalent API calls

```bash
# 1. Get metadata (includes content URL)
curl -u EMAIL:TOKEN "https://monitproduct.atlassian.net/rest/api/3/attachment/<attachment-id>"
# 2. Download via content URL from metadata response
curl -u EMAIL:TOKEN -L "<content-url>" -o output-file
```

### delete

Deletes an attachment by ID.

**Note:** No `-o json` branch — output is always the presenter text.

#### Default (table)

```
Deleted attachment 15974
```

#### Error: attachment not found

```
deleting attachment: deleting attachment 15974: resource not found: The attachment with id '15974' does not exist
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN -X DELETE "https://monitproduct.atlassian.net/rest/api/3/attachment/<attachment-id>"
```
## Global Flags

### `--verbose`

Shows request/response debug lines on stderr.

```
→ POST https://monitproduct.atlassian.net/rest/api/3/search/jql
← 200 OK
KEY | SUMMARY | STATUS | ASSIGNEE | TYPE
MON-4810 | Audit and remediate accessibility issues on Cap... | In Code Review | Aaron Wong | SDLC
More results available (use --next-page-token to fetch next page)
```

### `--no-color`

Disables ANSI escape sequences in table output.

```
KEY | SUMMARY | STATUS | ASSIGNEE | TYPE
MON-4810 | Audit and remediate accessibility issues on Cap... | In Code Review | Aaron Wong | SDLC
More results available (use --next-page-token to fetch next page)
```

### `--full -o json` — Agent vs Full Artifact Shapes

`--full` changes the JSON artifact shape. Below are the full-mode outputs for each command that uses `ArtifactMode()`.

#### me

Agent fields: `accountId`, `displayName`
Full adds: `email`, `active`

```json
{
  "accountId": "60e09bae7fcd820073089249",
  "displayName": "Rian Stockbower",
  "email": "rian@monitapp.io",
  "active": true
}
```

#### users get

Agent fields: `accountId`, `displayName`
Full adds: `email`, `active`

```json
{
  "accountId": "60e09bae7fcd820073089249",
  "displayName": "Rian Stockbower",
  "email": "rian@monitapp.io",
  "active": true
}
```

#### users search

Agent fields: `accountId`, `displayName`
Full adds: `email`, `active`

```json
{
  "results": [
    {
      "accountId": "60e09bae7fcd820073089249",
      "displayName": "Rian Stockbower",
      "email": "rian@monitapp.io",
      "active": true
    }
  ],
  "_meta": {
    "count": 1,
    "hasMore": false
  }
}
```

#### issues get

Agent fields: `key`, `summary`, `status`, `type`, `assignee`
Full adds: `priority`, `project`, `created`, `updated`, `reporter`, `labels`, `description`

```json
{
  "key": "MON-4810",
  "summary": "Audit and remediate accessibility issues on CapOne-specific surfaces",
  "status": "In Code Review",
  "type": "SDLC",
  "assignee": "Aaron Wong",
  "priority": "Medium",
  "project": "MON",
  "created": "2026-04-16",
  "updated": "2026-04-16",
  "reporter": "Aaron Wong",
  "description": "\nSummary\nPerform an accessibility-focused review and remediation pass across CapOne-specific frontend surfaces in packages/legacy/app, then validate the highest-risk interaction patterns.\nPrimary audit artifact:\n- docs/capone-accessibility-audit-2026-04-15.md\n\n\nProblem\nThe CapOne-specific UI surfaces are in mixed shape from an accessibility standpoint. Source review found a few meaningful issues concentrated in loading/redirect status communication, interactive semantics, and modal/illustration behavior.\nThe biggest findings are:\n- loading / redirect surfaces do not expose accessible status updates\n- CaponeStepsPreview uses tooltip semantics for interactive content\n- CaponeUnsupportedPackageModal is missing aria-describedby\n- some splash / package-option images are likely decorative but may be over-announced\n- CapOne-specific a11y coverage is currently light for keyboard / SR-focused behavior\n\n\nIn Scope\nCapOne-specific surfaces including:\n- packages/legacy/app/caponeLoadingScreen.ts\n- packages/legacy/app/containers/CaponeRedirect/CaponeRedirect.tsx\n- packages/legacy/app/fi-experiences/capone/CaponeTopBanner.tsx\n- packages/legacy/app/fi-experiences/capone/CaponeErrorPage.tsx\n- packages/legacy/app/fi-experiences/capone/CaponeSmbSplash.tsx\n- packages/legacy/app/fi-experiences/capone/CaponeStepsPreview.tsx\n- packages/legacy/app/fi-experiences/capone/CaponeUnsupportedPackageModal.tsx\n- packages/legacy/app/fi-experiences/capone/CaponeAccountingPackageOption.tsx\n\n\nProposed Work\n\n1. Loading / redirect accessibility\n- add accessible loading/status messaging to the CapOne loading screen and redirect/loading surfaces\n- ensure users of assistive tech receive meaningful progress/state feedback during SSO transfer and redirect waiting states\n\n\n2. Interactive semantics cleanup\n- replace tooltip semantics in the CapOne steps preview flow with a more appropriate interactive pattern for content that contains controls\n- improve trigger labeling/state where needed\n\n\n3. Modal relationship fix\n- add aria-describedby support for the unsupported-package modal body content\n\n\n4. Decorative / redundant image announcement review\n- review CapOne splash imagery and package-option logos\n- mark decorative images as decorative where appropriate\n- avoid duplicate accessible naming where visible text already names the option/content\n\n\n5. Validation / coverage\n- add or update targeted tests for the highest-confidence fixes\n- run manual keyboard + screen-reader-focused validation on the main CapOne surfaces\n\n\nAcceptance Criteria\n- CapOne loading and redirect states expose meaningful accessible status to assistive technologies\n- CaponeStepsPreview no longer exposes interactive content as a tooltip-style surface\n- the unsupported-package modal exposes both title and body text correctly to assistive tech\n- decorative CapOne imagery is not unnecessarily announced\n- package-option controls do not produce redundant or confusing accessible names\n- CapOne-specific keyboard and screen-reader validation is documented for the remediated surfaces\n\n\nNotes\nThis ticket intentionally keeps all CapOne accessibility findings in one place for now, rather than splitting into multiple implementation tickets.\n"
}
```

#### issues list

Agent fields: `key`, `summary`, `status`, `type`, `assignee`
Full adds: `priority`, `project`, `created`, `updated`, `reporter`, `labels`, `description`

```json
{
  "results": [
    {
      "key": "MON-4810",
      "summary": "Audit and remediate accessibility issues on CapOne-specific surfaces",
      "status": "In Code Review",
      "type": "SDLC",
      "assignee": "Aaron Wong",
      "priority": "Medium",
      "project": "MON",
      "created": "2026-04-16",
      "updated": "2026-04-16",
      "reporter": "Aaron Wong",
      "description": "\nSummary\nPerform an accessibility-focused review and remediation pass across CapOne-specific frontend surfaces in packages/legacy/app, then validate the highest-risk interaction patterns.\nPrimary audit artifact:\n- docs/capone-accessibility-audit-2026-04-15.md\n\n\nProblem\nThe CapOne-specific UI surfaces are in mixed shape from an accessibility standpoint. Source review found a few meaningful issues concentrated in loading/redirect status communication, interactive semantics, and modal/illustration behavior.\nThe biggest findings are:\n- loading / redirect surfaces do not expose accessible status updates\n- CaponeStepsPreview uses tooltip semantics for interactive content\n- CaponeUnsupportedPackageModal is missing aria-describedby\n- some splash / package-option images are likely decorative but may be over-announced\n- CapOne-specific a11y coverage is currently light for keyboard / SR-focused behavior\n\n\nIn Scope\nCapOne-specific surfaces including:\n- packages/legacy/app/caponeLoadingScreen.ts\n- packages/legacy/app/containers/CaponeRedirect/CaponeRedirect.tsx\n- packages/legacy/app/fi-experiences/capone/CaponeTopBanner.tsx\n- packages/legacy/app/fi-experiences/capone/CaponeErrorPage.tsx\n- packages/legacy/app/fi-experiences/capone/CaponeSmbSplash.tsx\n- packages/legacy/app/fi-experiences/capone/CaponeStepsPreview.tsx\n- packages/legacy/app/fi-experiences/capone/CaponeUnsupportedPackageModal.tsx\n- packages/legacy/app/fi-experiences/capone/CaponeAccountingPackageOption.tsx\n\n\nProposed Work\n\n1. Loading / redirect accessibility\n- add accessible loading/status messaging to the CapOne loading screen and redirect/loading surfaces\n- ensure users of assistive tech receive meaningful progress/state feedback during SSO transfer and redirect waiting states\n\n\n2. Interactive semantics cleanup\n- replace tooltip semantics in the CapOne steps preview flow with a more appropriate interactive pattern for content that contains controls\n- improve trigger labeling/state where needed\n\n\n3. Modal relationship fix\n- add aria-describedby support for the unsupported-package modal body content\n\n\n4. Decorative / redundant image announcement review\n- review CapOne splash imagery and package-option logos\n- mark decorative images as decorative where appropriate\n- avoid duplicate accessible naming where visible text already names the option/content\n\n\n5. Validation / coverage\n- add or update targeted tests for the highest-confidence fixes\n- run manual keyboard + screen-reader-focused validation on the main CapOne surfaces\n\n\nAcceptance Criteria\n- CapOne loading and redirect states expose meaningful accessible status to assistive technologies\n- CaponeStepsPreview no longer exposes interactive content as a tooltip-style surface\n- the unsupported-package modal exposes both title and body text correctly to assistive tech\n- decorative CapOne imagery is not unnecessarily announced\n- package-option controls do not produce redundant or confusing accessible names\n- CapOne-specific keyboard and screen-reader validation is documented for the remediated surfaces\n\n\nNotes\nThis ticket intentionally keeps all CapOne accessibility findings in one place for now, rather than splitting into multiple implementation tickets.\n"
    },
    {
      "key": "MON-4807",
      "summary": "Make CapOne key-stack authoritative for zero-state back behavior",
      "status": "In Code Review",
      "type": "SDLC",
      "assignee": "Aaron Wong",
      "priority": "Medium",
      "project": "MON",
      "created": "2026-04-15",
      "updated": "2026-04-16",
      "reporter": "Aaron Wong",
      "description": "\nProblem\nThe CapOne top-banner Back button previously relied on two different zero-state signals:\n- history.state.idx relative to a recorded baseIdx\n- a settled-navigation depth heuristic\n\nThose signals could disagree after successful in-app back navigation, especially around onboarding flows and full-page navigations. We also observed a concrete failure mode after refresh on a non-root page: baseIdx could be re-recorded at the current index while the preserved nav stack still showed real in-app depth, causing premature exit under the old OR-gate.\nThe implementation now fixes that by making the settled key-stack authoritative for zero-state detection. baseIdx / byIdx are retained for diagnostics only during rollout and no longer affect the live exit decision.\nRemaining uncertainty is no longer the stack algorithm itself; it is validating that real CapOne onboarding route sequences map to the intended semantic session root in partner-like environments.\n\nChange\nReplace the old monotonic settled-nav counter with a settled key-stack of { pathname, key } entries driven by React Router navigation semantics.\n\nStack rules\n- PUSH -\u003e append entry\n- REPLACE -\u003e swap top entry, no depth change\n- POP -\u003e truncate to matching key if known; otherwise append and treat as forward navigation\n\n\nZero-state decision\n- isZeroState = stackDepth \u003c= 1\n- the key-stack is the single source of truth\n- baseIdx / byIdx remain logged for diagnostics only\n\n\nPersistence / boot behavior\nPersist the stack to sessionStorage so depth survives soft reloads.\nOn the hasStoredTokens() boot path:\n- preserve stack for history-preserving navigations:\n  - navigation.type === \"reload\"\n  - navigation.type === \"back_forward\"\n\n- reset stack for fresh navigations:\n  - navigation.type === \"navigate\"\n\n\nThis preserves in-app back behavior after refresh / browser history restore, while still clearing stale depth on OAuth / external-return style full-page navigations.\n\nScope\nThis ticket hardens zero-state/back-state tracking and boot/reset behavior. It does not change the partner exit-target policy itself.\n\nSuccess Criteria\n\nFunctional\n- Zero-state is determined by key-stack depth, not baseIdx\n- byIdx=true must not cause exit when stackDepth \u003e 1\n- Refresh on a non-root settled page does not prematurely exit\n- Browser back/forward history restore preserves usable in-app back behavior\n- OAuth / external-return full-page navigation clears stale stack state before the next settled page\n\n\nOnboarding flow\n- In the CapOne onboarding flow, expected behavior is:\n  - deeper settled page (for example, integration) -\u003e Back navigates in-app\n  - effective onboarding root -\u003e Back exits Monit\n\n\n\nDiagnostics / rollout validation\n- byIdx / baseIdx remain present in logs for comparison, but are diagnostic-only\n- handleBack decision logs clearly show:\n  - route at click\n  - stackDepth\n  - byNav\n  - byIdx\n  - chosen branch\n\n- Real xbx / partner-like validation confirms the observed onboarding route sequence produces the intended stack depth and back behavior\n\n\nFiles\n- packages/legacy/app/fi-experiences/capone/caponeSession.ts\n- packages/legacy/app/fi-experiences/capone/CaponeTopBanner.tsx\n- packages/legacy/app/app.jsx\n- packages/legacy/app/fi-experiences/capone/caponeSession.test.ts\n- packages/legacy/app/fi-experiences/capone/caponeNavStack.test.ts\n- packages/legacy/app/fi-experiences/capone/caponeNavigation.test.ts\n- packages/legacy/app/fi-experiences/capone/CaponeTopBanner.test.tsx\n- packages/legacy/e2e/tests/smoke.capone.spec.ts\n\n"
    },
    {
      "key": "MON-4809",
      "summary": "Bump PostHog sampling to 100% for CapOne sessions",
      "status": "Backlog",
      "type": "SDLC",
      "priority": "Medium",
      "project": "MON",
      "created": "2026-04-16",
      "updated": "2026-04-16",
      "reporter": "Aaron Wong",
      "description": "\nProblem\nCapOne sessions are subject to the default 10% PostHog sampling rate (DEFAULT_SAMPLE_PERCENT = 10 in analytics.ts). CapOne pipes our PostHog analytics to their own dashboards — missing 90% of events will cause confusion and incomplete reporting on their side.\nPostHog session recording is also not enabled for CapOne (only Positive Pay gets startSessionRecording).\n\nComplexity\nThis is NOT a one-line change. The sampling decision is made once in initializeAnalyticsForRoute() and shared across DD RUM and PostHog (correlated sampling). At that point, the route is a generic /secure/* or /onboarding/* path — same as every other tenant.\nTo scope 100% sampling to CapOne only, the function needs a tenant signal. Options:\n- shouldBlockKeycloakInitForDomain() — available at call time (domain-based, no async), but couples analytics to auth config. Currently only CapOne uses blocked-keycloak domains, so this works today but is fragile.\n- isCapOneTenant() from utils/tenants.ts — cleaner, but requires the FI ID which comes from either domain config or Keycloak token. Need to verify it is available before initializeAnalyticsForRoute() runs.\n- Route-based check for /capone-sso — only works on the initial STS entry, not on refresh/OAuth return where the path is /secure/dashboard.\n\nWhichever approach is used, the correlated sampling (sampled flag shared between DD RUM and PostHog) means bumping PostHog to 100% for CapOne also bumps DD RUM to 100%. This may or may not be desired — if not, the sampling paths need to be decoupled for CapOne.\n\nFiles\n- packages/legacy/app/utils/analytics.ts — initializeAnalyticsForRoute(), initializePostHogForRoute(), shouldSampleSession()\n- packages/legacy/app/utils/tenants.ts — isCapOneTenant() predicate\n- packages/legacy/app/utils/domainConfig.ts — shouldBlockKeycloakInitForDomain()\n\n\nContext\nDiscovered during MON-4807 (CapOne back-button key-stack rewrite). DD Logs covers 100% of CapOne sessions and is our primary debugging tool, but the PostHog gap affects CapOne's own analytics consumption.\n[MON-4807]\n"
    }
  ],
  "_meta": {
    "count": 3,
    "hasMore": true
  }
}
```

#### issues search

Agent fields: `key`, `summary`, `status`, `type`, `assignee`
Full adds: `priority`, `project`, `created`, `updated`, `reporter`, `labels`, `description`

```json
{
  "results": [
    {
      "key": "MON-4810",
      "summary": "Audit and remediate accessibility issues on CapOne-specific surfaces",
      "status": "In Code Review",
      "type": "SDLC",
      "assignee": "Aaron Wong",
      "priority": "Medium",
      "project": "MON",
      "created": "2026-04-16",
      "updated": "2026-04-16",
      "reporter": "Aaron Wong",
      "description": "\nSummary\nPerform an accessibility-focused review and remediation pass across CapOne-specific frontend surfaces in packages/legacy/app, then validate the highest-risk interaction patterns.\nPrimary audit artifact:\n- docs/capone-accessibility-audit-2026-04-15.md\n\n\nProblem\nThe CapOne-specific UI surfaces are in mixed shape from an accessibility standpoint. Source review found a few meaningful issues concentrated in loading/redirect status communication, interactive semantics, and modal/illustration behavior.\nThe biggest findings are:\n- loading / redirect surfaces do not expose accessible status updates\n- CaponeStepsPreview uses tooltip semantics for interactive content\n- CaponeUnsupportedPackageModal is missing aria-describedby\n- some splash / package-option images are likely decorative but may be over-announced\n- CapOne-specific a11y coverage is currently light for keyboard / SR-focused behavior\n\n\nIn Scope\nCapOne-specific surfaces including:\n- packages/legacy/app/caponeLoadingScreen.ts\n- packages/legacy/app/containers/CaponeRedirect/CaponeRedirect.tsx\n- packages/legacy/app/fi-experiences/capone/CaponeTopBanner.tsx\n- packages/legacy/app/fi-experiences/capone/CaponeErrorPage.tsx\n- packages/legacy/app/fi-experiences/capone/CaponeSmbSplash.tsx\n- packages/legacy/app/fi-experiences/capone/CaponeStepsPreview.tsx\n- packages/legacy/app/fi-experiences/capone/CaponeUnsupportedPackageModal.tsx\n- packages/legacy/app/fi-experiences/capone/CaponeAccountingPackageOption.tsx\n\n\nProposed Work\n\n1. Loading / redirect accessibility\n- add accessible loading/status messaging to the CapOne loading screen and redirect/loading surfaces\n- ensure users of assistive tech receive meaningful progress/state feedback during SSO transfer and redirect waiting states\n\n\n2. Interactive semantics cleanup\n- replace tooltip semantics in the CapOne steps preview flow with a more appropriate interactive pattern for content that contains controls\n- improve trigger labeling/state where needed\n\n\n3. Modal relationship fix\n- add aria-describedby support for the unsupported-package modal body content\n\n\n4. Decorative / redundant image announcement review\n- review CapOne splash imagery and package-option logos\n- mark decorative images as decorative where appropriate\n- avoid duplicate accessible naming where visible text already names the option/content\n\n\n5. Validation / coverage\n- add or update targeted tests for the highest-confidence fixes\n- run manual keyboard + screen-reader-focused validation on the main CapOne surfaces\n\n\nAcceptance Criteria\n- CapOne loading and redirect states expose meaningful accessible status to assistive technologies\n- CaponeStepsPreview no longer exposes interactive content as a tooltip-style surface\n- the unsupported-package modal exposes both title and body text correctly to assistive tech\n- decorative CapOne imagery is not unnecessarily announced\n- package-option controls do not produce redundant or confusing accessible names\n- CapOne-specific keyboard and screen-reader validation is documented for the remediated surfaces\n\n\nNotes\nThis ticket intentionally keeps all CapOne accessibility findings in one place for now, rather than splitting into multiple implementation tickets.\n"
    },
    {
      "key": "MON-4809",
      "summary": "Bump PostHog sampling to 100% for CapOne sessions",
      "status": "Backlog",
      "type": "SDLC",
      "priority": "Medium",
      "project": "MON",
      "created": "2026-04-16",
      "updated": "2026-04-16",
      "reporter": "Aaron Wong",
      "description": "\nProblem\nCapOne sessions are subject to the default 10% PostHog sampling rate (DEFAULT_SAMPLE_PERCENT = 10 in analytics.ts). CapOne pipes our PostHog analytics to their own dashboards — missing 90% of events will cause confusion and incomplete reporting on their side.\nPostHog session recording is also not enabled for CapOne (only Positive Pay gets startSessionRecording).\n\nComplexity\nThis is NOT a one-line change. The sampling decision is made once in initializeAnalyticsForRoute() and shared across DD RUM and PostHog (correlated sampling). At that point, the route is a generic /secure/* or /onboarding/* path — same as every other tenant.\nTo scope 100% sampling to CapOne only, the function needs a tenant signal. Options:\n- shouldBlockKeycloakInitForDomain() — available at call time (domain-based, no async), but couples analytics to auth config. Currently only CapOne uses blocked-keycloak domains, so this works today but is fragile.\n- isCapOneTenant() from utils/tenants.ts — cleaner, but requires the FI ID which comes from either domain config or Keycloak token. Need to verify it is available before initializeAnalyticsForRoute() runs.\n- Route-based check for /capone-sso — only works on the initial STS entry, not on refresh/OAuth return where the path is /secure/dashboard.\n\nWhichever approach is used, the correlated sampling (sampled flag shared between DD RUM and PostHog) means bumping PostHog to 100% for CapOne also bumps DD RUM to 100%. This may or may not be desired — if not, the sampling paths need to be decoupled for CapOne.\n\nFiles\n- packages/legacy/app/utils/analytics.ts — initializeAnalyticsForRoute(), initializePostHogForRoute(), shouldSampleSession()\n- packages/legacy/app/utils/tenants.ts — isCapOneTenant() predicate\n- packages/legacy/app/utils/domainConfig.ts — shouldBlockKeycloakInitForDomain()\n\n\nContext\nDiscovered during MON-4807 (CapOne back-button key-stack rewrite). DD Logs covers 100% of CapOne sessions and is our primary debugging tool, but the PostHog gap affects CapOne's own analytics consumption.\n[MON-4807]\n"
    },
    {
      "key": "MON-4808",
      "summary": "Support deep-link tab navigation in Q2 campaign CTA URLs",
      "status": "Backlog",
      "type": "SDLC",
      "priority": "Medium",
      "project": "MON",
      "created": "2026-04-16",
      "updated": "2026-04-16",
      "reporter": "Aaron Wong",
      "description": "\nProblem\nCampaign CTA buttons in Q2 can set ctaButtonUrl to a Q2 entrypoint name (e.g., Main, Insights). However, entrypoints like Insights and Benchmarking are small widgets — opening them as a full Tecton drawer looks wrong. Currently Main is the only appropriate full-drawer entrypoint.\n\nSolution\nSupport fragment syntax in ctaButtonUrl (e.g., Main#insights, Main#benchmarking) so that:\n- The CTA opens the Main entrypoint (full dashboard drawer)\n- The fragment is passed as queryParams: { tab: \"\u003cfragment\u003e\" } via the Tecton SDK navigateTo API\n- DashboardTabs already reads ?tab= from the URL, so the correct tab activates automatically\n\n\nChanges required\nsignal-webapp-frontend only (no admin app changes needed — ctaButtonUrl is already a free-text input with no validation):\n- useMonitTectonNavHandler.ts — update handleOpenEntrypoint to parse #fragment and pass as queryParams\n- Q2CampaignPromotions.tsx — pass the raw ctaButtonUrl (including fragment) to updated handler\n\n\nTechnical detail\nThe Q2 Tecton SDK navigateTo supports queryParams natively:\n\nnavigateTo(extensionName, \"Main\", { tab: \"insights\" })\nQuery params flow through to the URL and are picked up by DashboardTabs via useLocation().search.\n\nValid tab values\nkey-numbers, insights, events, benchmarking (from TabOptions enum)\n"
    }
  ],
  "_meta": {
    "count": 3,
    "hasMore": true
  }
}
```

#### boards list

Agent fields: `id`, `name`, `type`
Full adds: `projectKey`, `projectName`

```json
{
  "results": [
    {
      "id": 26,
      "name": "OFF board",
      "type": "simple",
      "projectKey": "OFF",
      "projectName": "On/Offboarding"
    },
    {
      "id": 27,
      "name": "INCIDENT board",
      "type": "simple",
      "projectKey": "INCIDENT",
      "projectName": "Incidents"
    },
    {
      "id": 24,
      "name": "ON board",
      "type": "kanban",
      "projectKey": "ON",
      "projectName": "Customer Onboarding"
    },
    {
      "id": 23,
      "name": "MON board",
      "type": "scrum",
      "projectKey": "MON",
      "projectName": "Platform Development"
    },
    {
      "id": 25,
      "name": "JAR board",
      "type": "kanban",
      "projectKey": "JAR",
      "projectName": "Jira Application Requests"
    },
    {
      "id": 12,
      "name": "OP board",
      "type": "kanban",
      "projectKey": "OP",
      "projectName": "Operations"
    },
    {
      "id": 28,
      "name": "TST board",
      "type": "scrum"
    }
  ],
  "_meta": {
    "count": 7,
    "hasMore": false
  }
}
```

#### boards get

Agent fields: `id`, `name`, `type`
Full adds: `projectKey`, `projectName`

```json
{
  "id": 23,
  "name": "MON board",
  "type": "scrum",
  "projectKey": "MON",
  "projectName": "Platform Development"
}
```

#### sprints list

Agent fields: `id`, `name`, `state`
Full adds: `startDate`, `endDate`, `completeDate`, `goal`, `boardId`

```json
{
  "results": [
    {
      "id": 58,
      "name": "MON Sprint 1",
      "state": "closed",
      "startDate": "2023-08-14T15:48:59Z",
      "endDate": "2023-08-25T15:48:00Z",
      "completeDate": "2023-09-28T03:38:32Z",
      "goal": "Originally Created: 2023-08-14T18:17:46.438Z - Originally Completed: 2023-08-28T13:34:02.407Z",
      "boardId": 23
    },
    {
      "id": 59,
      "name": "MON Sprint 2",
      "state": "closed",
      "startDate": "2023-08-28T00:00:35Z",
      "endDate": "2023-09-08T23:30:00Z",
      "completeDate": "2023-09-28T03:39:13Z",
      "goal": "Originally Created: 2023-08-28T13:34:02.476Z - Originally Completed: 2023-09-11T18:09:07.741Z",
      "boardId": 23
    },
    {
      "id": 60,
      "name": "MON Sprint 3",
      "state": "closed",
      "startDate": "2023-09-11T18:10:36Z",
      "endDate": "2023-09-22T00:00:00Z",
      "completeDate": "2023-09-28T03:40:29Z",
      "goal": "Originally Created: 2023-08-28T15:13:42.793Z - Originally Completed: 2023-09-25T15:26:39.706Z",
      "boardId": 23
    },
    {
      "id": 61,
      "name": "MON Sprint 4",
      "state": "closed",
      "startDate": "2023-09-25T15:26:04Z",
      "endDate": "2023-10-06T00:00:00Z",
      "completeDate": "2023-10-11T15:18:32Z",
      "goal": "Originally Created: 2023-09-12T15:52:49.843Z",
      "boardId": 23
    },
    {
      "id": 62,
      "name": "MON Sprint 5",
      "state": "closed",
      "startDate": "2023-10-11T15:18:07Z",
      "endDate": "2023-10-20T00:00:00Z",
      "completeDate": "2023-10-20T21:45:45Z",
      "boardId": 23
    },
    {
      "id": 63,
      "name": "MON Sprint 6",
      "state": "closed",
      "startDate": "2023-10-23T15:00:44Z",
      "endDate": "2023-11-03T23:42:00Z",
      "completeDate": "2023-11-07T15:27:47Z",
      "boardId": 23
    },
    {
      "id": 64,
      "name": "MON Sprint 7",
      "state": "closed",
      "startDate": "2023-11-06T15:27:06Z",
      "endDate": "2023-11-17T00:00:00Z",
      "completeDate": "2023-11-20T17:19:33Z",
      "boardId": 23
    },
    {
      "id": 65,
      "name": "MON Sprint 8",
      "state": "closed",
      "startDate": "2023-11-20T17:20:12Z",
      "endDate": "2023-12-01T01:53:01Z",
      "completeDate": "2023-12-02T19:34:24Z",
      "boardId": 23
    },
    {
      "id": 66,
      "name": "MON Sprint 9",
      "state": "closed",
      "startDate": "2023-12-04T16:01:58Z",
      "endDate": "2023-12-15T01:53:01Z",
      "completeDate": "2023-12-18T19:08:21Z",
      "boardId": 23
    },
    {
      "id": 67,
      "name": "MON Sprint 10",
      "state": "closed",
      "startDate": "2023-12-18T19:08:44Z",
      "endDate": "2024-01-05T01:53:00Z",
      "completeDate": "2024-01-08T15:58:21Z",
      "boardId": 23
    },
    {
      "id": 68,
      "name": "MON Sprint 11",
      "state": "closed",
      "startDate": "2024-01-08T15:58:50Z",
      "endDate": "2024-01-19T22:43:00Z",
      "completeDate": "2024-01-22T17:15:27Z",
      "boardId": 23
    },
    {
      "id": 69,
      "name": "MON Sprint 12",
      "state": "closed",
      "startDate": "2024-01-22T17:15:54Z",
      "endDate": "2024-02-02T00:00:00Z",
      "completeDate": "2024-02-17T01:20:44Z",
      "boardId": 23
    },
    {
      "id": 70,
      "name": "MON Sprint 14",
      "state": "closed",
      "startDate": "2024-02-19T01:21:00Z",
      "endDate": "2024-03-01T08:06:00Z",
      "completeDate": "2024-03-04T23:12:54Z",
      "boardId": 23
    },
    {
      "id": 71,
      "name": "MON Sprint 15",
      "state": "closed",
      "startDate": "2024-03-04T23:13:24Z",
      "endDate": "2024-03-15T05:58:00Z",
      "completeDate": "2024-03-30T00:11:57Z",
      "boardId": 23
    },
    {
      "id": 72,
      "name": "MON Sprint 17",
      "state": "closed",
      "startDate": "2024-04-01T00:15:00Z",
      "endDate": "2024-04-12T07:00:00Z",
      "completeDate": "2024-04-12T21:49:51Z",
      "boardId": 23
    },
    {
      "id": 73,
      "name": "MON Sprint 18",
      "state": "closed",
      "startDate": "2024-04-15T03:00:00Z",
      "endDate": "2024-04-26T20:00:00Z",
      "completeDate": "2024-04-26T23:55:13Z",
      "boardId": 23
    },
    {
      "id": 74,
      "name": "MON Sprint 19",
      "state": "closed",
      "startDate": "2024-04-29T15:02:27Z",
      "endDate": "2024-05-10T00:00:00Z",
      "completeDate": "2024-05-13T15:14:48Z",
      "boardId": 23
    },
    {
      "id": 75,
      "name": "MON Sprint 20",
      "state": "closed",
      "startDate": "2024-05-13T15:19:39Z",
      "endDate": "2024-05-24T00:00:00Z",
      "completeDate": "2024-05-25T00:08:50Z",
      "boardId": 23
    },
    {
      "id": 76,
      "name": "MON Sprint 21",
      "state": "closed",
      "startDate": "2024-05-27T00:09:00Z",
      "endDate": "2024-06-07T00:00:00Z",
      "completeDate": "2024-06-10T14:58:02Z",
      "boardId": 23
    },
    {
      "id": 77,
      "name": "MON Sprint 22",
      "state": "closed",
      "startDate": "2024-06-10T14:58:04Z",
      "endDate": "2024-06-21T00:00:00Z",
      "completeDate": "2024-06-26T15:02:34Z",
      "boardId": 23
    },
    {
      "id": 78,
      "name": "MON Sprint 23",
      "state": "closed",
      "startDate": "2024-06-24T15:02:07Z",
      "endDate": "2024-07-05T00:00:00Z",
      "completeDate": "2024-07-08T14:55:43Z",
      "boardId": 23
    },
    {
      "id": 79,
      "name": "MON Sprint 24",
      "state": "closed",
      "startDate": "2024-07-08T14:55:04Z",
      "endDate": "2024-07-19T00:00:00Z",
      "completeDate": "2024-08-01T23:05:25Z",
      "boardId": 23
    },
    {
      "id": 80,
      "name": "MON Sprint 25",
      "state": "closed",
      "startDate": "2024-07-19T23:06:19Z",
      "endDate": "2024-08-02T00:00:00Z",
      "completeDate": "2024-08-07T10:29:04Z",
      "boardId": 23
    },
    {
      "id": 81,
      "name": "MON Sprint 26",
      "state": "closed",
      "startDate": "2024-08-07T10:29:18Z",
      "endDate": "2024-08-16T00:00:00Z",
      "completeDate": "2024-08-17T00:05:20Z",
      "boardId": 23
    },
    {
      "id": 82,
      "name": "MON Sprint 27",
      "state": "closed",
      "startDate": "2024-08-19T00:05:00Z",
      "endDate": "2024-08-30T00:00:00Z",
      "completeDate": "2024-09-03T14:59:34Z",
      "boardId": 23
    },
    {
      "id": 83,
      "name": "MON Sprint 28",
      "state": "closed",
      "startDate": "2024-09-03T14:59:50Z",
      "endDate": "2024-09-13T00:00:00Z",
      "completeDate": "2024-09-16T17:42:26Z",
      "boardId": 23
    },
    {
      "id": 84,
      "name": "MON Sprint 29",
      "state": "closed",
      "startDate": "2024-09-16T17:42:49Z",
      "endDate": "2024-09-27T00:00:00Z",
      "completeDate": "2024-09-30T14:59:24Z",
      "boardId": 23
    },
    {
      "id": 85,
      "name": "MON Sprint 30",
      "state": "closed",
      "startDate": "2024-09-30T14:59:43Z",
      "endDate": "2024-10-11T00:00:00Z",
      "completeDate": "2024-10-16T15:13:10Z",
      "boardId": 23
    },
    {
      "id": 86,
      "name": "MON Sprint 31",
      "state": "closed",
      "startDate": "2024-10-14T15:13:45Z",
      "endDate": "2024-10-25T00:00:00Z",
      "completeDate": "2024-10-28T18:19:43Z",
      "boardId": 23
    },
    {
      "id": 87,
      "name": "MON Sprint 32",
      "state": "closed",
      "startDate": "2024-10-28T18:19:01Z",
      "endDate": "2024-11-08T00:00:00Z",
      "completeDate": "2024-11-08T19:02:30Z",
      "boardId": 23
    },
    {
      "id": 88,
      "name": "MON Sprint 33",
      "state": "closed",
      "startDate": "2024-11-11T19:02:00Z",
      "endDate": "2024-11-22T19:02:00Z",
      "completeDate": "2024-11-22T23:34:29Z",
      "boardId": 23
    },
    {
      "id": 89,
      "name": "MON Sprint 34",
      "state": "closed",
      "startDate": "2024-11-22T23:34:44Z",
      "endDate": "2024-12-06T00:00:00Z",
      "completeDate": "2024-12-06T21:22:11Z",
      "boardId": 23
    },
    {
      "id": 90,
      "name": "MON Sprint 35",
      "state": "closed",
      "startDate": "2024-12-09T21:22:00Z",
      "endDate": "2024-12-20T00:00:00Z",
      "completeDate": "2025-01-02T21:16:51Z",
      "boardId": 23
    },
    {
      "id": 91,
      "name": "MON Sprint 36",
      "state": "closed",
      "startDate": "2025-01-03T00:00:47Z",
      "endDate": "2025-01-04T00:00:00Z",
      "completeDate": "2025-01-06T16:02:39Z",
      "boardId": 23
    },
    {
      "id": 92,
      "name": "MON Sprint 37",
      "state": "closed",
      "startDate": "2025-01-06T16:02:59Z",
      "endDate": "2025-01-17T00:00:00Z",
      "completeDate": "2025-01-21T16:19:18Z",
      "boardId": 23
    },
    {
      "id": 93,
      "name": "MON Sprint 38",
      "state": "closed",
      "startDate": "2025-01-21T16:20:09Z",
      "endDate": "2025-01-31T00:00:00Z",
      "completeDate": "2025-01-31T16:55:23Z",
      "boardId": 23
    },
    {
      "id": 94,
      "name": "MON Sprint 39",
      "state": "closed",
      "startDate": "2025-02-03T16:55:00Z",
      "endDate": "2025-02-14T16:55:00Z",
      "completeDate": "2025-02-18T15:59:29Z",
      "boardId": 23
    },
    {
      "id": 95,
      "name": "MON Sprint 40",
      "state": "closed",
      "startDate": "2025-02-18T15:59:50Z",
      "endDate": "2025-02-28T00:00:00Z",
      "completeDate": "2025-03-03T15:57:45Z",
      "boardId": 23
    },
    {
      "id": 96,
      "name": "MON Sprint 41",
      "state": "closed",
      "startDate": "2025-03-03T15:58:28Z",
      "endDate": "2025-03-14T00:00:00Z",
      "completeDate": "2025-03-18T15:12:29Z",
      "boardId": 23
    },
    {
      "id": 97,
      "name": "MON Sprint 42",
      "state": "closed",
      "startDate": "2025-03-17T15:12:54Z",
      "endDate": "2025-03-28T00:00:00Z",
      "completeDate": "2025-04-01T17:09:58Z",
      "boardId": 23
    },
    {
      "id": 98,
      "name": "MON Sprint 43",
      "state": "closed",
      "startDate": "2025-03-31T17:12:36Z",
      "endDate": "2025-04-11T00:00:00Z",
      "completeDate": "2025-04-22T18:39:10Z",
      "boardId": 23
    },
    {
      "id": 99,
      "name": "MON Sprint 44",
      "state": "closed",
      "startDate": "2025-04-14T18:39:00Z",
      "endDate": "2025-04-25T00:00:00Z",
      "completeDate": "2025-04-28T21:53:54Z",
      "boardId": 23
    },
    {
      "id": 100,
      "name": "MON Sprint 45",
      "state": "closed",
      "startDate": "2025-04-28T21:54:15Z",
      "endDate": "2025-05-09T00:00:00Z",
      "completeDate": "2025-05-12T15:35:33Z",
      "boardId": 23
    },
    {
      "id": 101,
      "name": "MON Sprint 46",
      "state": "closed",
      "startDate": "2025-05-12T15:35:49Z",
      "endDate": "2025-05-23T00:00:00Z",
      "completeDate": "2025-05-27T15:05:00Z",
      "boardId": 23
    },
    {
      "id": 102,
      "name": "MON Sprint 47",
      "state": "closed",
      "startDate": "2025-05-27T15:05:23Z",
      "endDate": "2025-06-06T00:00:00Z",
      "completeDate": "2025-06-27T15:30:35Z",
      "boardId": 23
    },
    {
      "id": 103,
      "name": "MON Sprint 48",
      "state": "closed",
      "startDate": "2025-06-23T15:30:56Z",
      "endDate": "2025-07-04T00:00:00Z",
      "completeDate": "2025-07-08T17:10:34Z",
      "boardId": 23
    },
    {
      "id": 104,
      "name": "MON Sprint 50",
      "state": "closed",
      "startDate": "2025-07-07T15:03:04Z",
      "endDate": "2025-07-18T00:00:00Z",
      "completeDate": "2025-07-21T15:52:20Z",
      "boardId": 23
    },
    {
      "id": 106,
      "name": "MON Sprint 51",
      "state": "closed",
      "startDate": "2025-07-22T14:15:30Z",
      "endDate": "2025-08-01T23:30:00Z",
      "completeDate": "2025-08-05T19:30:43Z",
      "boardId": 23
    },
    {
      "id": 107,
      "name": "MON Sprint 52",
      "state": "closed",
      "startDate": "2025-08-04T00:00:17Z",
      "endDate": "2025-08-15T23:30:00Z",
      "completeDate": "2025-08-18T18:17:43Z",
      "boardId": 23
    },
    {
      "id": 108,
      "name": "MON Sprint 53",
      "state": "closed",
      "startDate": "2025-08-18T18:17:54Z",
      "endDate": "2025-08-29T00:00:00Z",
      "completeDate": "2025-09-08T15:02:27Z",
      "boardId": 23
    }
  ],
  "_meta": {
    "count": 50,
    "hasMore": true
  }
}
```

#### sprints current

Agent fields: `id`, `name`, `state`
Full adds: `startDate`, `endDate`, `completeDate`, `goal`, `boardId`

```json
{
  "id": 125,
  "name": "MON Sprint 70",
  "state": "active",
  "startDate": "2026-04-10T00:00:45Z",
  "endDate": "2026-04-24T23:30:00Z",
  "boardId": 23
}
```

#### sprints issues

Agent fields: `key`, `summary`, `status`, `type`, `assignee`
Full adds: `priority`, `project`, `created`, `updated`, `reporter`, `labels`, `description`

```json
{
  "results": [
    {
      "key": "MON-1918",
      "summary": "Q2 and Banno Notifications for MSR send",
      "status": "In Development",
      "type": "SDLC",
      "assignee": "Devin Quirk",
      "priority": "Medium",
      "project": "MON",
      "created": "2024-10-30",
      "updated": "2026-04-10",
      "reporter": "Rusty Hall",
      "description": "During research of notifications through Q2 it looks like the Caliper API must be used. We need to research this feature to see what the integration with that api looks like and how it compares to something similar on the Banno end. \n\n[https://docs.google.com/document/d/1u-btsm27HTkWMg9IuroA8Dm7yKY421oaLHnkKeVQJQM/edit?tab=t.0|https://docs.google.com/document/d/1u-btsm27HTkWMg9IuroA8Dm7yKY421oaLHnkKeVQJQM/edit?tab=t.0|smart-link] "
    },
    {
      "key": "MON-2846",
      "summary": "Add Funnel Chart \u0026 Interactive Stage Behavior to Clients Page",
      "status": "Backlog",
      "type": "SDLC",
      "assignee": "Devin Quirk",
      "priority": "Medium",
      "project": "MON",
      "created": "2025-07-24",
      "updated": "2026-04-10",
      "reporter": "Rusty Hall",
      "description": "*Title*: Implement Interactive Funnel Chart with Stage Detail Integration\n\nh4. 🧩 Overview:\n\nIntroduce a new funnel visualization to represent the client journey across four activation stages. Include interactivity (hover and click), visual feedback for table updates, and integration with the Stage Details panel.\n\n----\n\nh4. 🔹 Funnel Chart Requirements:\n\n*Position*:\n\n* Top-left of the _Accounting Connected Clients_ page, above the client table.\n\n*Stages* (from left to right):\n\n# *Started Activation*\n# *Landed at Integration*\n# *Selected Package*\n# *Activated Users*\n\n*Visual Notes*:\n\n* Funnel widths should visually represent the relative number of clients in each stage.\n* Each stage section should have a subtle gradient fill (as shown in the design).\n\nh4. Hover Interaction:\n\n* When the user hovers over any stage:\n** That stage displays a *white outline stroke* to indicate hover state.\n** Rows in the client table below that match the hovered stage are *highlighted* with a *light blue background* (e.g. {{#E5F0FF}}).\n** Tooltip or label is optional unless required by UX—no popover is expected at this time.\n\n----\n\nh4. 🔹 Click Interaction:\n\n* Clicking a stage in the funnel:\n** Locks the stage selection (\"click-to-stick\").\n** Selected stage *remains outlined* until user clicks elsewhere or clears selection.\n** Table remains *filtered to only show clients* in that stage.\n** Highlighting of filtered rows remains consistent with hover behavior.\n* Only one stage may be selected at a time.\n\n----\n\nh4. 🔹 Stage Details Panel:\n\n*Position*:\n\n* To the right of the funnel chart.\n\n*Content*:\n\n* Table showing counts for each stage and the *\"View\"* link per row.\n* On clicking the {{?}} icon, expand the box below with the following stage definitions \n* Stage 1: Includes all users who started the activation process.  \nStage 2: Includes all users who made it to the integration page.  \nStage 3: Includes all users who selected a supported accounting package.  \nStage 4: Includes all users who are activated.\n\nh4. 🔹 \"View\" Button Behavior:\n\n* Clicking *\"View\"* on a row in the Stage Details table:\n** Filters the client table to the selected stage.\n** Triggers a *front-end visual effect* (such as a brief *fade-in animation* or *highlight pulse*) to indicate that the table has updated.\n** Funnel stage should also update to reflect selected stage (i.e., same as if the user clicked the funnel).\n\nh4. 🧪 Edge/UX Notes:\n\n* If the user clicks an already-selected funnel stage or View button again, keep it selected (no toggle behavior).\n* Add a “Clear Selection” button or UI affordance (optional) to reset all filters.\n\n*Design:* \n\n[https://www.figma.com/design/GdKh1e8doN9DhVB2RHrDLU/Monit---Master-File?node-id=10688-40596\u0026t=LbAKeZ9YLfFVELiQ-1|https://www.figma.com/design/GdKh1e8doN9DhVB2RHrDLU/Monit---Master-File?node-id=10688-40596\u0026t=LbAKeZ9YLfFVELiQ-1|smart-link] "
    },
    {
      "key": "MON-3151",
      "summary": "Add additional identifiers to insights and actions and audience builder exports",
      "status": "Ready for Development",
      "type": "SDLC",
      "assignee": "Devin Quirk",
      "priority": "Medium",
      "project": "MON",
      "created": "2025-10-22",
      "updated": "2026-04-10",
      "reporter": "Rusty Hall",
      "description": "As noted from the message below from Fairwinds credit union, there are cases that the *“external id”*  and the “*fi business id”* may be used by an FI. We need to get both in all exports from the banker portal. I’ve noted what’s needed to get that done in all the exports:\n\n-Add FI Business Id id to insights and actions export (for example, the FI business Id column from the Clients page export). \n\n-Add external user id to audience builder (for example, the external id on the Clients page export)."
    }
  ],
  "_meta": {
    "count": 3,
    "hasMore": true
  }
}
```

#### comments list

Agent fields: `id`, `author`, `created`, `body` (truncated at 200 chars)
Full adds: `updated`, `body` (untruncated)

```json
{
  "results": [
    {
      "id": "21242",
      "author": "Aaron Wong",
      "created": "2026-04-16",
      "body": "Short audit conclusion after the current code changes:\nThe major source-level accessibility findings on CapOne-specific surfaces appear to be addressed or materially improved:\n- loading / redirect states now expose accessible status messaging\n- the unsupported-package modal now exposes both title and description correctly\n- step triggers now have stronger names/state semantics\n- reviewed decorative imagery is no longer over-announced where visible text already carries the meaning\n\nThe main remaining risk is no longer an obvious code-level defect. It is runtime validation of the interactive step-preview surface and live-region behavior against real keyboard / assistive-technology behavior.\nPractical readout:\n- Resolved or materially improved at code level: loading status, modal description wiring, weak trigger labels, redundant decorative image announcements\n- Still requires manual validation: CaponeStepsPreview interaction/focus behavior, loading announcement behavior across real SR/browser combinations, and rendered focus visibility\n\nBottom line: this now looks much stronger from a source audit perspective; remaining uncertainty is primarily manual conformance validation, not missing obvious ARIA wiring.",
      "updated": "2026-04-16"
    }
  ],
  "_meta": {
    "count": 1,
    "hasMore": false
  }
}
```

#### automation get

Agent fields: `id`, `name`, `state`, `componentSummary`
Full adds: `description`, `labels`, `tags`

```json
{
  "id": "018c2840-57c1-7869-9393-11205cc87ce4",
  "name": "ON/MON: Create Onboarding Tasks",
  "state": "ENABLED",
  "componentSummary": "27 total — 4 condition(s), 23 action(s)",
  "description": "Creates Tasks when a new Onboarding Epic is created"
}
```
## Command Aliases

### Top-level aliases

All aliases produce identical output to their canonical form. Verified outputs:

#### `jtk issue` / `jtk i` → `jtk issues`

```
KEY | SUMMARY | STATUS | ASSIGNEE | TYPE
MON-4810 | Audit and remediate accessibility issues on Cap... | In Code Review | Aaron Wong | SDLC
More results available (use --next-page-token to fetch next page)
```

#### `jtk project` / `jtk proj` / `jtk p` → `jtk projects`

```
KEY | NAME | TYPE | LEAD
INCIDENT | Incidents | software | 
```

#### `jtk board` / `jtk b` → `jtk boards`

```
ID | NAME | TYPE | PROJECT
26 | OFF board | simple | OFF
```

#### `jtk sprint` / `jtk sp` → `jtk sprints`

```
ID | NAME | STATE | START | END
125 | MON Sprint 70 | active | 2026-04-10 | 2026-04-24
```

#### `jtk user` / `jtk u` → `jtk users`

```
ACCOUNT ID | NAME | EMAIL | ACTIVE
60e09bae7fcd820073089249 | Rian Stockbower | rian@monitapp.io | yes
```

#### `jtk auto` → `jtk automation`

```
UUID | NAME | STATE
018c2840-57c1-7869-9393-11205cc87ce4 | ON/MON: Create Onboarding Tasks | ENABLED
```

#### `jtk transition` / `jtk tr` → `jtk transitions`

```
ID | NAME | TO STATUS
121 | Ready to QA | Ready for QA
151 | Return to Development | In Development
11 | Backlog | Backlog
21 | Ready for Development | Ready for Development
31 | In Development | In Development
41 | In Code Review | In Code Review
51 | Ready for QA | Ready for QA
61 | Ready for Deployment | Ready for Deployment
71 | Deployed | Deployed
81 | Canceled | Canceled
```

#### `jtk comment` / `jtk c` → `jtk comments`

```
ID | AUTHOR | CREATED | BODY
21242 | Aaron Wong | 2026-04-16 | Short audit conclusion after the current code changes: The major source-level accessibility findings... [truncated, use --no-truncate for complete text]
```

#### `jtk attachment` / `jtk att` → `jtk attachments`

```
No attachments found on MON-4810
```

#### `jtk field` / `jtk f` → `jtk fields`

```
unknown flag: --max
```

#### `jtk link` / `jtk l` → `jtk links`

```
No links on MON-4810
```

#### `jtk dash` / `jtk dashboard` → `jtk dashboards`

```
ID | NAME | OWNER | FAVOURITE
10000 | Default dashboard |  | no
```

### Subcommand aliases

#### `jtk attachments ls` → `jtk attachments list`

```
No attachments found on MON-4810
```

#### `jtk attachments rm` → `jtk attachments delete`

```
deleting attachment: deleting attachment 99999: resource not found: The attachment with id '99999' does not exist
```

#### `jtk fields ctx` / `jtk fields context` → `jtk fields contexts`

```
ID | NAME | GLOBAL | ANY_ISSUE_TYPE
10139 | Default Configuration Scheme for Banking Platform | yes | yes
10178 | Engineering | no | yes
```

#### `jtk fields opt` / `jtk fields option` → `jtk fields options`

```
ID | VALUE | DISABLED
10020 | Apiture | no
10021 | Banno | no
10022 | Narmi | no
10023 | Q2 | no
10030 | No Platform | no
```
## Mutations

All write commands executed against live Jira instance. Test data cleaned up.

### issues create / update / assign / transition / delete

#### Create

```
Created issue MON-4811 (https://monitproduct.atlassian.net/browse/MON-4811)
```

#### Get (verify creation)

```
Key: MON-4811
Summary: [Test] Integration Test Issue
Status: Backlog
Type: SDLC
Priority: Medium
Assignee: Unassigned
Project: MON
URL: https://monitproduct.atlassian.net/browse/MON-4811
```

#### Update description

```
Updated issue MON-4811
```

#### Update summary

```
Updated issue MON-4811
```

#### Assign

```
Assigned issue MON-4811 to Rian Stockbower
```

#### comments add

```
Added comment 21275 to MON-4811
```

#### comments list

```
ID | AUTHOR | CREATED | BODY
21275 | Rian Stockbower | 2026-04-16 | Line oneLine twoIndented line 
```

#### comments list --no-truncate

```
ID: 21275
Author: Rian Stockbower
Created: 2026-04-16
Body: Line oneLine twoIndented line
```

#### transitions list --fields

```
ID | NAME | TO STATUS | REQUIRED FIELDS
91 | Ready | Ready for Development | -
161 | Start | In Development | -
11 | Backlog | Backlog | -
21 | Ready for Development | Ready for Development | -
31 | In Development | In Development | -
41 | In Code Review | In Code Review | -
51 | Ready for QA | Ready for QA | -
61 | Ready for Deployment | Ready for Deployment | -
71 | Deployed | Deployed | -
81 | Canceled | Canceled | -
```

#### transitions do

```
Transitioned MON-4811
```

#### issues assign --unassign

```
Unassigned issue MON-4811
```

#### issues update --assignee none

```
Updated issue MON-4811
```

#### comments delete

```
Deleted comment 21275 from MON-4811
```

#### Create with multi-value `--field` (customfield_10044 = Meta Status)

```
Created issue MON-4812 (https://monitproduct.atlassian.net/browse/MON-4812)
```

#### issues delete --force

```
Deleted issue MON-4811
Deleted issue MON-4812
```

#### Error cases

```
required flag(s) "summary" not set
required flag(s) "project" not set
updating issue MON-99999: resource not found: Issue does not exist or you do not have permission to see it.
deleting issue MON-99999: resource not found: Issue does not exist or you do not have permission to see it.
```

#### Equivalent API calls

```bash
# Create
curl -u EMAIL:TOKEN -X POST "BASE/rest/api/3/issue" \
  -H "Content-Type: application/json" \
  -d '{"fields":{"project":{"key":"MON"},"issuetype":{"name":"SDLC"},"summary":"..."}}'
# Update
curl -u EMAIL:TOKEN -X PUT "BASE/rest/api/3/issue/MON-4811" \
  -H "Content-Type: application/json" \
  -d '{"fields":{"summary":"...","description":{"type":"doc","version":1,"content":[...]}}}'
# Assign
curl -u EMAIL:TOKEN -X PUT "BASE/rest/api/3/issue/MON-4811/assignee" \
  -H "Content-Type: application/json" \
  -d '{"accountId":"60e09bae7fcd820073089249"}'
# Transition
curl -u EMAIL:TOKEN -X POST "BASE/rest/api/3/issue/MON-4811/transitions" \
  -H "Content-Type: application/json" \
  -d '{"transition":{"id":"91"}}'
# Delete
curl -u EMAIL:TOKEN -X DELETE "BASE/rest/api/3/issue/MON-4811"
```

### links create / delete

#### Create two issues

```
Created issue MON-4813 (https://monitproduct.atlassian.net/browse/MON-4813)
Created issue MON-4814 (https://monitproduct.atlassian.net/browse/MON-4814)
```

#### links create

```
Created Blocker link: MON-4813 → MON-4814
```

#### links list (with link ID)

```
ID | TYPE | DIRECTION | ISSUE | SUMMARY
17843 | Blocker | blocks | MON-4814 | [Test] Link Target
```

#### links delete

```
Deleted link 17843
```

#### links list after delete

```
No links on MON-4813
```

#### Error cases

```
resource not found: Issue does not exist or you do not have permission to see it.
link type "NonexistentType" not found (available: Blocker, Cloners, Duplicate, Polaris work item link, Relates)
resource not found: No issue link with id '99999' exists.
```

#### Equivalent API calls

```bash
# Create link
curl -u EMAIL:TOKEN -X POST "BASE/rest/api/3/issueLink" \
  -H "Content-Type: application/json" \
  -d '{"type":{"name":"Blocker"},"outwardIssue":{"key":"MON-4813"},"inwardIssue":{"key":"MON-4814"}}'
# Delete link
curl -u EMAIL:TOKEN -X DELETE "BASE/rest/api/3/issueLink/17843"
```

### projects create / update / delete / restore

#### Create

```
Created project ZTEST (Integration Test Project)
```

#### Get

```
Key: ZTEST
Name: Integration Test Project
ID: 10068
Type: software
Lead: Rian Stockbower
Issue Types: [Task Sub-task]
```

#### Update

```
Updated project ZTEST
```

#### Get (verify update)

```
Key: ZTEST
Name: Updated Test Project
ID: 10068
Type: software
Lead: Rian Stockbower
Issue Types: [Task Sub-task]
```

#### Delete (soft-delete to trash)

```
Deleted project ZTEST (moved to trash)
```

#### Restore

```
Restored project ZTEST (Updated Test Project)
```

#### Final delete (cleanup)

```
Deleted project ZTEST (moved to trash)
```

#### Error cases

```
required flag(s) "lead", "name" not set
fetching project: resource not found: No project could be found with key 'NONEXISTENT'.
```

#### Equivalent API calls

```bash
# Create
curl -u EMAIL:TOKEN -X POST "BASE/rest/api/3/project" \
  -H "Content-Type: application/json" \
  -d '{"key":"ZTEST","name":"...","projectTypeKey":"software","leadAccountId":"..."}'
# Update
curl -u EMAIL:TOKEN -X PUT "BASE/rest/api/3/project/ZTEST" \
  -H "Content-Type: application/json" -d '{"name":"Updated Test Project"}'
# Delete (soft)
curl -u EMAIL:TOKEN -X DELETE "BASE/rest/api/3/project/ZTEST"
# Restore
curl -u EMAIL:TOKEN -X POST "BASE/rest/api/3/project/ZTEST/restore"
```

### dashboards create / get / delete

#### Create

```
Created dashboard [Test] Integration Dashboard (10071)
URL: /jira/dashboards/10071
```

#### Get

```
ID: 10071
Name: [Test] Integration Dashboard
Owner: Rian Stockbower
URL: /jira/dashboards/10071
```

#### List with search

```
ID | NAME | OWNER | FAVOURITE
10071 | [Test] Integration Dashboard |  | no
10000 | Default dashboard |  | no
10001 | Epics |  | no
```

#### gadgets list (empty)

```
No gadgets on dashboard 10071
```

#### Delete

```
Deleted dashboard 10071
```

#### Get after delete (404)

```
resource not found: The dashboard with id '10071' does not exist.
```

#### Error cases

```
required flag(s) "name" not set
resource not found: The dashboard with id '99999' does not exist.
```

#### Equivalent API calls

```bash
# Create
curl -u EMAIL:TOKEN -X POST "BASE/rest/api/3/dashboard" \
  -H "Content-Type: application/json" \
  -d '{"name":"[Test] Integration Dashboard","sharePermissions":[]}'
# Delete
curl -u EMAIL:TOKEN -X DELETE "BASE/rest/api/3/dashboard/10071"
```

### automation create / enable / disable / update / delete

#### Export source rule

```bash
jtk automation export 018c2840-57c1-7869-9393-11205cc87ce4 > source.json
```

#### Create copy (strip UUID, rename)

```
Created automation rule (UUID: 019d959b-99ec-7c4b-a670-a67ae08b44ae)
```

#### Get (verify)

```
Name: [Test] Auto Integration Copy
UUID: 019d959b-99ec-7c4b-a670-a67ae08b44ae
State: ENABLED
Description: Creates Tasks when a new Onboarding Epic is created
Components: 27 total — 4 condition(s), 23 action(s)
```

#### disable

```
Rule "[Test] Auto Integration Copy": ENABLED → DISABLED
```

#### enable

```
Rule "[Test] Auto Integration Copy": DISABLED → ENABLED
```

#### enable (idempotent)

```
Rule "[Test] Auto Integration Copy" is already ENABLED
```

#### Round-trip update

```
Updated automation rule 019d959b-99ec-7c4b-a670-a67ae08b44ae
Updating rule: [Test] Auto Integration Copy (UUID: 019d959b-99ec-7c4b-a670-a67ae08b44ae, State: ENABLED)
```

#### Delete

```
Deleted automation rule "[Test] Auto Integration Copy" (019d959b-99ec-7c4b-a670-a67ae08b44ae)
```

#### Error cases

```
required flag(s) "file" not set
getting automation rule 99999999: resource not found
```

### sprints add

#### Create test issue

```
Created issue MON-4815 (https://monitproduct.atlassian.net/browse/MON-4815)
```

#### Add to sprint 125 (MON Sprint 70)

```
Moved MON-4815 to sprint 125
```

#### Equivalent API call

```bash
curl -u EMAIL:TOKEN -X POST "BASE/rest/agile/1.0/sprint/125/issue" \
  -H "Content-Type: application/json" \
  -d '{"issues":["MON-4815"]}'
```

### fields create / contexts / options / delete / restore

#### Create field

```
Created field customfield_10222 ([Test] Integration Select)
```

#### List to verify

```
No fields found
```

#### contexts list

```
ID | NAME | GLOBAL | ANY_ISSUE_TYPE
10397 | Default Configuration Scheme for [Test] Integration Select | yes | yes
```

#### options add

```
Added option Option A
Added option Option B
```

#### options list

```
ID | VALUE | DISABLED
10110 | Option A | no
10111 | Option B | no
```

#### options update

```
Updated option 10110
```

#### options list after update

```
ID | VALUE | DISABLED
10110 | Option A (updated) | no
10111 | Option B | no
```

#### options delete

```
Deleted option 10110 from field customfield_10222
```

#### contexts create (project-scoped)

```
Created context 10398 ([Test] Context)
```

#### contexts delete

```
Deleted context 10398 from field customfield_10222
```

#### fields delete (trash)

```
Trashed field customfield_10222
```

#### fields restore

```
Restored field customfield_10222
```

#### fields delete again (final cleanup)

```
Trashed field customfield_10222
```

#### Error cases

```
required flag(s) "name", "type" not set
trashing field customfield_99999: resource not found: Field not found.
fetching field contexts: resource not found: The custom field was not found.
```

#### Equivalent API calls

```bash
# Create field
curl -u EMAIL:TOKEN -X POST "BASE/rest/api/3/field" \
  -H "Content-Type: application/json" \
  -d '{"name":"[Test] Integration Select","type":"com.atlassian.jira.plugin.system.customfieldtypes:select"}'
# Add option
curl -u EMAIL:TOKEN -X POST "BASE/rest/api/3/field/customfield_10222/context/10397/option" \
  -H "Content-Type: application/json" \
  -d '{"options":[{"value":"Option A"}]}'
# Trash field
curl -u EMAIL:TOKEN -X DELETE "BASE/rest/api/3/field/customfield_10222"
# Restore field
curl -u EMAIL:TOKEN -X PUT "BASE/rest/api/3/field/customfield_10222/restore"
```

## Mutation JSON branches

Some write commands emit different JSON when called with `-o json`. These were not captured in the main mutation lifecycles above.

### comments add `-o json`

Returns the full raw `api.Comment` object (including ADF body structure).

```json
{
  "id": "21276",
  "author": {
    "accountId": "60e09bae7fcd820073089249",
    "displayName": "Rian Stockbower",
    "emailAddress": "rian@monitapp.io",
    "active": true,
    "avatarUrls": {
      "16x16": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/60e09bae7fcd820073089249/724b7268-43a2-4a44-a00e-7a204fe99f90/16",
      "24x24": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/60e09bae7fcd820073089249/724b7268-43a2-4a44-a00e-7a204fe99f90/24",
      "32x32": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/60e09bae7fcd820073089249/724b7268-43a2-4a44-a00e-7a204fe99f90/32",
      "48x48": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/60e09bae7fcd820073089249/724b7268-43a2-4a44-a00e-7a204fe99f90/48"
    }
  },
  "body": {
    "type": "doc",
    "version": 1,
    "content": [
      {
        "type": "paragraph",
        "content": [
          {
            "type": "text",
            "text": "JSON branch test Thu Apr 16 05:56:22 EDT 2026"
          }
        ]
      }
    ]
  },
  "created": "2026-04-16T09:56:22.447+0000",
  "updated": "2026-04-16T09:56:22.447+0000"
}
```

### comments delete `-o json`

```json
{
  "commentId": "21276",
  "status": "deleted"
}
```

### links create `-o json`

Note: the link ID is **not** returned. To get the link ID after creation, use `links list <issue-key> -o json`.

```json
{
  "inwardIssue": "MON-4819",
  "outwardIssue": "MON-4818",
  "status": "created",
  "type": "Blocker"
}
```

### links delete `-o json`

```json
{
  "linkId": "17844",
  "status": "deleted"
}
```

### projects create `-o json`

Returns the created project. Note: `name` appears empty in the API response immediately after creation — it populates asynchronously.

```json
{
  "id": 10069,
  "key": "GFIL",
  "name": ""
}
```

### automation delete `-o json`

```json
{
  "name": "[Test] GapFill Automation Delete",
  "ruleId": "019d95ba-031c-7000-88df-134a1c924860",
  "status": "deleted"
}
```

### dashboards gadgets remove `-o json`

```json
{
  "dashboardId": "10072",
  "gadgetId": 10122,
  "status": "removed"
}
```

## Mutation JSON branches (round 2)

Additional write commands with `-o json` branches not captured in round 1.

### issues create `-o json`

Returns the raw `api.Issue` from the Jira API. Note: `summary` appears empty immediately after creation.

```json
{
  "id": "34487",
  "key": "MON-4820",
  "self": "https://monitproduct.atlassian.net/rest/api/3/issue/34487",
  "fields": {
    "summary": ""
  }
}
```

### projects update `-o json`

Returns the full `api.Project` object after update.

```json
{
  "id": 10070,
  "key": "JR2F",
  "name": "[Test] JSON Round2 Updated",
  "projectTypeKey": "software",
  "lead": {
    "accountId": "60e09bae7fcd820073089249",
    "displayName": "Rian Stockbower",
    "active": true,
    "avatarUrls": {
      "16x16": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/60e09bae7fcd820073089249/724b7268-43a2-4a44-a00e-7a204fe99f90/16",
      "24x24": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/60e09bae7fcd820073089249/724b7268-43a2-4a44-a00e-7a204fe99f90/24",
      "32x32": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/60e09bae7fcd820073089249/724b7268-43a2-4a44-a00e-7a204fe99f90/32",
      "48x48": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/60e09bae7fcd820073089249/724b7268-43a2-4a44-a00e-7a204fe99f90/48"
    }
  },
  "issueTypes": [
    {
      "id": "10007",
      "name": "Task",
      "description": "A small, distinct piece of work.",
      "subtask": false
    },
    {
      "id": "10008",
      "name": "Sub-task",
      "description": "A small piece of work that's part of a larger task.",
      "subtask": true
    }
  ]
}
```

### projects restore `-o json`

Returns the restored project (fewer fields than update).

```json
{
  "id": 10070,
  "key": "JR2F",
  "name": "[Test] JSON Round2 Updated",
  "projectTypeKey": "software"
}
```

### dashboards create `-o json`

Returns the full `api.Dashboard` object.

```json
{
  "id": "10073",
  "name": "[Test] JSON dash",
  "owner": {
    "accountId": "60e09bae7fcd820073089249",
    "displayName": "Rian Stockbower",
    "active": true,
    "avatarUrls": {
      "16x16": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/60e09bae7fcd820073089249/724b7268-43a2-4a44-a00e-7a204fe99f90/16",
      "24x24": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/60e09bae7fcd820073089249/724b7268-43a2-4a44-a00e-7a204fe99f90/24",
      "32x32": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/60e09bae7fcd820073089249/724b7268-43a2-4a44-a00e-7a204fe99f90/32",
      "48x48": "https://avatar-management--avatars.us-west-2.prod.public.atl-paas.net/60e09bae7fcd820073089249/724b7268-43a2-4a44-a00e-7a204fe99f90/48"
    }
  },
  "view": "/jira/dashboards/10073",
  "isFavourite": true
}
```

### dashboards delete `-o json`

```json
{
  "dashboardId": "10073",
  "status": "deleted"
}
```

### fields create `-o json`

Returns the full `api.Field` object including `clauseNames` for JQL.

```json
{
  "id": "customfield_10223",
  "key": "customfield_10223",
  "name": "[Test] JSON Select",
  "custom": true,
  "orderable": true,
  "navigable": true,
  "searchable": true,
  "schema": {
    "type": "option",
    "custom": "com.atlassian.jira.plugin.system.customfieldtypes:select",
    "customId": 10223
  },
  "clauseNames": [
    "[Test] JSON Select[Dropdown]",
    "cf[10223]",
    "[Test] JSON Select"
  ]
}
```

### fields contexts create `-o json`

```json
{
  "id": "10402",
  "name": "[Test] JSON Context",
  "isGlobalContext": false,
  "isAnyIssueType": false
}
```

### fields options add `-o json`

Returns `null` — the Atlassian API response uses an `"options"` key but jtk parses the `"values"` key, resulting in nil.

```json
null
```

### fields options update `-o json`

Same issue as `options add` — returns `null`.

```json
null
```
