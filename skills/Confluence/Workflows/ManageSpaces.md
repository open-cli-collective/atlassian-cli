# ManageSpaces Workflow

List, view, create, and update Confluence spaces.

## Intent-to-Flag Mapping

### Action Selection

| User Says | Command | Required |
|-----------|---------|----------|
| "list spaces", "show spaces" | `cfl space list` | Nothing |
| "global spaces only" | `cfl space list --type global` | Nothing |
| "personal spaces" | `cfl space list --type personal` | Nothing |
| "space details", "show space" | `cfl space view KEY` | Space key |
| "create space", "new space" | `cfl space create --key KEY --name "NAME"` | Key + name |
| "update space", "rename space" | `cfl space update KEY` | Space key + fields |

## Execute

### List Spaces

```bash
# All spaces
cfl space list

# Global spaces only
cfl space list --type global

# With higher limit
cfl space list --limit 50
```

### View Space Details

```bash
cfl space view SPACE_KEY
```

### Create Space

```bash
# Basic creation
cfl space create --key KEY --name "Space Name"

# With description
cfl space create --key KEY --name "Space Name" --description "Description text"
```

### Update Space

```bash
# Update name
cfl space update KEY --name "New Name"

# Update description
cfl space update KEY --description "New description"

# Update both
cfl space update KEY --name "New Name" --description "New description"
```

## Output Format

- **List:** Table with Key, Name, Type
- **View:** Full space details including key, name, type, description
- **Create:** Confirm creation with key and name
- **Update:** Confirm what was changed

## Post-Action

After any action:
1. For list: state total space count
2. For create: confirm space key and name, note it's ready for pages
3. For update: confirm what was changed
4. For view: show space details and offer to list pages in the space
