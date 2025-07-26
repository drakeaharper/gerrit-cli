#!/bin/bash

# Gerry CLI Installer
# This script installs gerry and ensures it's in your PATH

set -e

echo "🔧 Installing Gerry CLI..."

# Build and install
make clean
make build

# Force copy to ensure we get the latest binary
if [[ -w /usr/local/bin ]]; then
    echo "Installing to /usr/local/bin/gerry..."
    cp bin/gerry /usr/local/bin/gerry
    chmod +x /usr/local/bin/gerry
    INSTALL_PATH="/usr/local/bin/gerry"
else
    echo "Installing to ~/bin/gerry..."
    mkdir -p "$HOME/bin"
    cp bin/gerry "$HOME/bin/gerry"
    chmod +x "$HOME/bin/gerry"
    INSTALL_PATH="$HOME/bin/gerry"
fi

# Check if ~/bin is in PATH
if [[ ":$PATH:" != *":$HOME/bin:"* ]] && [[ -f "$HOME/bin/gerry" ]]; then
    echo ""
    echo "⚠️  ~/bin is not in your PATH. Adding it..."
    
    # Detect shell and add to appropriate profile
    if [[ "$SHELL" == *"zsh"* ]]; then
        PROFILE="$HOME/.zshrc"
    elif [[ "$SHELL" == *"bash"* ]]; then
        if [[ -f "$HOME/.bash_profile" ]]; then
            PROFILE="$HOME/.bash_profile"
        else
            PROFILE="$HOME/.bashrc"
        fi
    else
        PROFILE="$HOME/.profile"
    fi
    
    # Add to profile if not already there
    if ! grep -q 'export PATH="$HOME/bin:$PATH"' "$PROFILE" 2>/dev/null; then
        echo "" >> "$PROFILE"
        echo '# Added by gerry installer' >> "$PROFILE"
        echo 'export PATH="$HOME/bin:$PATH"' >> "$PROFILE"
        echo "✅ Added ~/bin to PATH in $PROFILE"
        echo "📝 Please run: source $PROFILE"
        echo "   Or restart your terminal"
    else
        echo "✅ PATH already configured in $PROFILE"
    fi
fi

echo ""
echo "🎉 Installation complete!"

# Test installation
echo "🧪 Testing installation..."
if command -v gerry >/dev/null 2>&1; then
    echo "✅ gerry is available in PATH"
    gerry version
    echo ""
    echo "Testing list command..."
    if gerry list --help >/dev/null 2>&1; then
        echo "✅ Commands are working correctly"
    else
        echo "⚠️  Commands may not be working. Try: hash -r"
    fi
else
    echo "⚠️  gerry not found in PATH. You may need to:"
    echo "   1. Restart your terminal, or"
    echo "   2. Run: source ~/.zshrc (or your shell profile)"
    echo "   3. Or add ~/bin to your PATH manually"
    echo "   4. Or run: hash -r"
fi

echo ""
echo "📖 Install man page? (y/n)"
read -r response
if [[ "$response" =~ ^[Yy]$ ]]; then
    echo "Installing man page..."
    make install-man
fi

echo ""
echo "🚀 Next steps:"
echo "   1. Run: gerry init"
echo "   2. Configure your Gerrit connection"
echo "   3. Start using: gerry list"
echo "   4. View help: man gerry (if man page installed)"