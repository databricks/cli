# The user, or another tool, adds a server of their own under the name that databricks is adding or removing.
jq '.mcpServers["github-logfood"] = {"command": "npx", "args": ["users-own-github-server"]}' home/.claude.json > home/claude.tmp
mv home/claude.tmp home/.claude.json
echo "user wrote their own github-logfood entry" >> concurrent.log
