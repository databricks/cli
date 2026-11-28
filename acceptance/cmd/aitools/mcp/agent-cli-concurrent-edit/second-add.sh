# A second databricks command, started while the first one waits for the claude CLI: the same server name for another
# profile.
$CLI aitools mcp add --profile dogfood --names system.ai.github --name github --agents claude-code &> second-add.txt || echo "Exit code: $?" >> second-add.txt
