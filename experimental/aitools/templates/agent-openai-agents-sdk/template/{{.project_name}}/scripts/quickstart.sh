#!/bin/bash
set -e

# Helper function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Helper function to check if Homebrew is available
has_brew() {
    command_exists brew
}

echo "==================================================================="
echo "Agent on Apps - Quickstart Setup"
echo "==================================================================="
echo

# ===================================================================
# Section 1: Prerequisites Installation
# ===================================================================

echo "Checking and installing prerequisites..."
echo

# Check and install UV
if command_exists uv; then
    echo "✓ UV is already installed"
    uv --version
else
    echo "Installing UV..."
    if has_brew; then
        echo "Using Homebrew to install UV..."
        brew install uv
    else
        echo "Using curl to install UV..."
        curl -LsSf https://astral.sh/uv/install.sh | sh
        # Add UV to PATH for current session
        export PATH="$HOME/.cargo/bin:$PATH"
    fi
    echo "✓ UV installed successfully"
fi

# Check and install nvm
if [ -s "$HOME/.nvm/nvm.sh" ]; then
    echo "✓ nvm is already installed"
    # Load nvm for current session
    export NVM_DIR="$HOME/.nvm"
    [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
else
    echo "Installing nvm..."
    if has_brew; then
        echo "Using Homebrew to install nvm..."
        brew install nvm
        # Create nvm directory
        mkdir -p ~/.nvm
        # Add nvm to current session
        export NVM_DIR="$HOME/.nvm"
        [ -s "/opt/homebrew/opt/nvm/nvm.sh" ] && \. "/opt/homebrew/opt/nvm/nvm.sh"
        [ -s "/usr/local/opt/nvm/nvm.sh" ] && \. "/usr/local/opt/nvm/nvm.sh"
    else
        echo "Using curl to install nvm..."
        curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash
        # Load nvm for current session
        export NVM_DIR="$HOME/.nvm"
        [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
    fi
    echo "✓ nvm installed successfully"
fi

# Use Node 20
echo "Setting up Node.js 20..."
nvm install 20
nvm use 20
echo "✓ Node.js 20 is now active"
node --version
npm --version
echo

# Check and install Databricks CLI
if command_exists databricks; then
    echo "✓ Databricks CLI is already installed"
    databricks --version
else
    echo "Installing Databricks CLI..."
    if has_brew; then
        echo "Using Homebrew to install Databricks CLI..."
        brew tap databricks/tap
        brew install databricks
    else
        echo "Using curl to install Databricks CLI..."
        if curl -fsSL https://raw.githubusercontent.com/databricks/setup-cli/main/install.sh | sh; then
            echo "✓ Databricks CLI installed successfully"
        else
            echo "Installation failed, trying with sudo..."
            curl -fsSL https://raw.githubusercontent.com/databricks/setup-cli/main/install.sh | sudo sh
        fi
    fi
    echo "✓ Databricks CLI installed successfully"
fi
echo

# ===================================================================
# Section 2: Configuration Files Setup
# ===================================================================
echo "Setting up configuration files..."

# Copy .env.example to .env.local if it doesn't exist
if [ ! -f ".env.local" ]; then
    echo "Copying .env.example to .env.local..."
    cp .env.example .env.local
    echo
else
    echo ".env.local already exists, skipping copy..."
fi
echo

# ===================================================================
# Section 3: Databricks Authentication
# ===================================================================

echo "Setting up Databricks authentication..."

# Check if there are existing profiles
set +e
EXISTING_PROFILES=$(databricks auth profiles 2>/dev/null)
PROFILES_EXIT_CODE=$?
set -e

if [ $PROFILES_EXIT_CODE -eq 0 ] && [ -n "$EXISTING_PROFILES" ]; then
    # Profiles exist - let user select one
    echo "Found existing Databricks profiles:"
    echo

    # Parse profiles into an array (compatible with older bash)
    # Skip the first line (header row)
    PROFILE_ARRAY=()
    PROFILE_NAMES=()
    LINE_NUM=0
    while IFS= read -r line; do
        if [ -n "$line" ]; then
            if [ $LINE_NUM -eq 0 ]; then
                # Print header without number
                echo "$line"
            else
                # Add full line to display array
                PROFILE_ARRAY+=("$line")
                # Extract just the profile name (first column) for selection
                PROFILE_NAME_ONLY=$(echo "$line" | awk '{print $1}')
                PROFILE_NAMES+=("$PROFILE_NAME_ONLY")
            fi
            LINE_NUM=$((LINE_NUM + 1))
        fi
    done <<< "$EXISTING_PROFILES"
    echo

    # Display numbered list
    for i in "${!PROFILE_ARRAY[@]}"; do
        echo "$((i+1))) ${PROFILE_ARRAY[$i]}"
    done
    echo

    echo "Enter the number of the profile you want to use:"
    read -r PROFILE_CHOICE

    if [ -z "$PROFILE_CHOICE" ]; then
        echo "Error: Profile selection is required"
        exit 1
    fi

    # Validate the choice is a number
    if ! [[ "$PROFILE_CHOICE" =~ ^[0-9]+$ ]]; then
        echo "Error: Please enter a valid number"
        exit 1
    fi

    # Convert to array index (subtract 1)
    PROFILE_INDEX=$((PROFILE_CHOICE - 1))

    # Check if the index is valid
    if [ $PROFILE_INDEX -lt 0 ] || [ $PROFILE_INDEX -ge ${#PROFILE_NAMES[@]} ]; then
        echo "Error: Invalid selection. Please choose a number between 1 and ${#PROFILE_NAMES[@]}"
        exit 1
    fi

    # Get the selected profile name (just the name, not the full line)
    PROFILE_NAME="${PROFILE_NAMES[$PROFILE_INDEX]}"
    echo "Selected profile: $PROFILE_NAME"

    # Test if the profile works
    set +e
    DATABRICKS_CONFIG_PROFILE="$PROFILE_NAME" databricks current-user me >/dev/null 2>&1
    PROFILE_TEST=$?
    set -e

    if [ $PROFILE_TEST -eq 0 ]; then
        echo "✓ Successfully validated profile '$PROFILE_NAME'"
    else
        # Profile exists but isn't authenticated - prompt to authenticate
        echo "Profile '$PROFILE_NAME' is not authenticated."
        echo "Authenticating profile '$PROFILE_NAME'..."
        echo "You will be prompted to log in to Databricks in your browser."
        echo

        # Temporarily disable exit on error for the auth command
        set +e

        # Run auth login with the profile name and capture output while still showing it to the user
        AUTH_LOG=$(mktemp)
        databricks auth login --profile "$PROFILE_NAME" 2>&1 | tee "$AUTH_LOG"
        AUTH_EXIT_CODE=$?

        set -e

        if [ $AUTH_EXIT_CODE -eq 0 ]; then
            echo "✓ Successfully authenticated profile '$PROFILE_NAME'"
            # Clean up temp file
            rm -f "$AUTH_LOG"
        else
            # Clean up temp file
            rm -f "$AUTH_LOG"
            echo "Error: Profile '$PROFILE_NAME' authentication failed"
            exit 1
        fi
    fi

    # Update .env.local with the profile name
    if grep -q "DATABRICKS_CONFIG_PROFILE=" .env.local; then
        sed -i '' "s/DATABRICKS_CONFIG_PROFILE=.*/DATABRICKS_CONFIG_PROFILE=$PROFILE_NAME/" .env.local
    else
        echo "DATABRICKS_CONFIG_PROFILE=$PROFILE_NAME" >> .env.local
    fi
    echo "✓ Databricks profile '$PROFILE_NAME' saved to .env.local"
else
    # No profiles exist - create default one
    echo "No existing profiles found. Setting up Databricks authentication..."
    echo "Please enter your Databricks host URL (e.g., https://your-workspace.cloud.databricks.com):"
    read -r DATABRICKS_HOST

    if [ -z "$DATABRICKS_HOST" ]; then
        echo "Error: Databricks host is required"
        exit 1
    fi

    echo "Authenticating with Databricks..."
    echo "You will be prompted to log in to Databricks in your browser."
    echo

    # Temporarily disable exit on error for the auth command
    set +e

    # Run auth login with host parameter and capture output while still showing it to the user
    AUTH_LOG=$(mktemp)
    databricks auth login --host "$DATABRICKS_HOST" 2>&1 | tee "$AUTH_LOG"
    AUTH_EXIT_CODE=$?

    set -e

    if [ $AUTH_EXIT_CODE -eq 0 ]; then
        echo "✓ Successfully authenticated with Databricks"

        # Extract profile name from the captured output
        # Expected format: "Profile DEFAULT was successfully saved"
        PROFILE_NAME=$(grep -i "Profile .* was successfully saved" "$AUTH_LOG" | sed -E 's/.*Profile ([^ ]+) was successfully saved.*/\1/' | head -1)

        # Clean up temp file
        rm -f "$AUTH_LOG"

        # If we couldn't extract the profile name, default to "DEFAULT"
        if [ -z "$PROFILE_NAME" ]; then
            PROFILE_NAME="DEFAULT"
            echo "Note: Could not detect profile name, using 'DEFAULT'"
        fi

        # Update .env.local with the profile name
        if grep -q "DATABRICKS_CONFIG_PROFILE=" .env.local; then
            sed -i '' "s/DATABRICKS_CONFIG_PROFILE=.*/DATABRICKS_CONFIG_PROFILE=$PROFILE_NAME/" .env.local
        else
            echo "DATABRICKS_CONFIG_PROFILE=$PROFILE_NAME" >> .env.local
        fi

        echo "✓ Databricks profile '$PROFILE_NAME' saved to .env.local"
    else
        # Clean up temp file
        rm -f "$AUTH_LOG"
        echo "Databricks authentication was cancelled or failed."
        echo "Please run this script again when you're ready to authenticate."
        exit 1
    fi
fi
echo

# ===================================================================
# Section 4: MLflow Experiment Setup
# ===================================================================


# Get current Databricks username
echo "Getting Databricks username..."
DATABRICKS_USERNAME=$(databricks -p $PROFILE_NAME current-user me | jq -r .userName)
echo "Username: $DATABRICKS_USERNAME"
echo

# Create MLflow experiment and capture the experiment ID
echo "Creating MLflow experiment..."
EXPERIMENT_NAME="/Users/$DATABRICKS_USERNAME/agents-on-apps"

# Try to create the experiment with the default name first
if EXPERIMENT_RESPONSE=$(databricks -p $PROFILE_NAME experiments create-experiment $EXPERIMENT_NAME 2>/dev/null); then
    EXPERIMENT_ID=$(echo $EXPERIMENT_RESPONSE | jq -r .experiment_id)
    echo "Created experiment '$EXPERIMENT_NAME' with ID: $EXPERIMENT_ID"
else
    echo "Experiment name already exists, creating with random suffix..."
    RANDOM_SUFFIX=$(openssl rand -hex 4)
    EXPERIMENT_NAME="/Users/$DATABRICKS_USERNAME/agents-on-apps-$RANDOM_SUFFIX"
    EXPERIMENT_RESPONSE=$(databricks -p $PROFILE_NAME experiments create-experiment $EXPERIMENT_NAME)
    EXPERIMENT_ID=$(echo $EXPERIMENT_RESPONSE | jq -r .experiment_id)
    echo "Created experiment '$EXPERIMENT_NAME' with ID: $EXPERIMENT_ID"
fi
echo

# Update .env.local with the experiment ID
echo "Updating .env.local with experiment ID..."
sed -i '' "s/MLFLOW_EXPERIMENT_ID=.*/MLFLOW_EXPERIMENT_ID=$EXPERIMENT_ID/" .env.local
echo

echo "==================================================================="
echo "Setup Complete!"
echo "==================================================================="
echo "✓ Prerequisites installed (UV, nvm, Databricks CLI)"
echo "✓ Databricks authenticated with profile: $PROFILE_NAME"
echo "✓ Configuration files created (.env.local)"
echo "✓ MLflow experiment created: $EXPERIMENT_NAME"
echo "✓ Experiment ID: $EXPERIMENT_ID"
echo "✓ Configuration updated in .env.local"
echo "==================================================================="
echo
