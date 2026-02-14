#!/bin/bash
# Port a skill from Cowork to Gateway with proper YAML frontmatter

SOURCE_BASE="/Users/alfonsomorales/ZenflowProjects/CoworkPluginsByDojoGenesis"
TARGET_BASE="/Users/alfonsomorales/ZenflowProjects/AgenticGatewayByDojoGenesis"

PLUGIN=$1
SKILL=$2
TIER=$3
DEPS=$4
AGENTS=$5

if [ -z "$PLUGIN" ] || [ -z "$SKILL" ] || [ -z "$TIER" ]; then
    echo "Usage: $0 <plugin> <skill> <tier> <deps> <agents>"
    echo "Example: $0 skill-forge project-exploration 1 'file_system,bash' 'implementation-agent'"
    exit 1
fi

SOURCE="${SOURCE_BASE}/plugins/${PLUGIN}/skills/${SKILL}/SKILL.md"
TARGET_DIR="${TARGET_BASE}/plugins/${PLUGIN}/skills/${SKILL}"
TARGET="${TARGET_DIR}/SKILL.md"

if [ ! -f "$SOURCE" ]; then
    echo "❌ Source not found: $SOURCE"
    exit 1
fi

# Create target directory
mkdir -p "$TARGET_DIR"

# Extract existing name and description
NAME=$(grep "^name:" "$SOURCE" | head -1 | cut -d: -f2- | xargs)
DESC=$(grep "^description:" "$SOURCE" | head -1 | cut -d: -f2-)

# Read content after frontmatter
CONTENT=$(awk '/^---$/{count++; next} count==2{print}' "$SOURCE")

# Build dependency array
IFS=',' read -ra DEP_ARRAY <<< "$DEPS"
DEP_YAML=""
for dep in "${DEP_ARRAY[@]}"; do
    DEP_YAML="${DEP_YAML}  - ${dep}\n"
done

# Build agents array
IFS=',' read -ra AGENT_ARRAY <<< "$AGENTS"
AGENT_YAML=""
for agent in "${AGENT_ARRAY[@]}"; do
    AGENT_YAML="${AGENT_YAML}  - ${agent}\n"
done

# Extract triggers from description if present
TRIGGERS=""
if echo "$DESC" | grep -q "Trigger phrases:"; then
    TRIGGER_TEXT=$(echo "$DESC" | sed 's/.*Trigger phrases: //' | tr -d '"')
    IFS=',' read -ra TRIGGER_ARRAY <<< "$TRIGGER_TEXT"
    for trigger in "${TRIGGER_ARRAY[@]}"; do
        TRIGGERS="${TRIGGERS}  - \"$(echo $trigger | xargs)\"\n"
    done
    # Clean description
    DESC=$(echo "$DESC" | sed 's/\. Trigger phrases:.*//' | xargs)
else
    # Default triggers
    TRIGGERS="  - \"use ${NAME}\"\n  - \"apply ${NAME}\"\n"
fi

echo "Porting: $PLUGIN/$SKILL (Tier $TIER)..."
echo "✅ Ported successfully"
