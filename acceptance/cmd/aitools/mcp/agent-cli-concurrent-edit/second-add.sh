# A second databricks command, started while the first one waits for the claude CLI: the same server name for another
# profile. It finishes before the first one writes.
$CLI aitools mcp add --profile dogfood --names system.ai.github --name github --agents claude-code &> second-add.txt
