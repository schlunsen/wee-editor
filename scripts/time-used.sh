#!/bin/bash

# Development Time Estimation Script
# Analyzes git commit history to estimate total development hours

set -e

# Default values
SESSION_GAP_HOURS=2
MIN_SESSION_MINUTES=15
MAX_SESSION_HOURS=8

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Function to print colored output
print_color() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Function to print section header
print_header() {
    echo
    print_color $CYAN "========================================"
    print_color $CYAN "$1"
    print_color $CYAN "========================================"
    echo
}

# Function to convert seconds to hours
seconds_to_hours() {
    local seconds=$1
    printf "%.2f" $(echo "scale=2; $seconds / 3600" | bc)
}

# Function to parse git log and calculate development time
calculate_development_time() {
    print_header "🔍 Analyzing Git History for Development Time"

    # Check if we're in a git repository
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        print_color $RED "❌ Not a git repository"
        exit 1
    fi

    # Get git log in format: timestamp|author|email|commit_hash|subject
    print_color $BLUE "📊 Parsing git commit history..."

    git log --pretty=format:"%at|%an|%ae|%H|%s" --reverse > /tmp/git_commits.txt

    # Debug: check if file was created
    if [[ ! -f /tmp/git_commits.txt ]]; then
        print_color $RED "❌ Failed to create git commits file"
        exit 1
    fi

    local total_commits=$(wc -l < /tmp/git_commits.txt)
    print_color $GREEN "✅ Found $total_commits commits"

    # Debug: show first few lines
    if [[ "$verbose" == "true" ]]; then
        print_color $YELLOW "🔍 First 5 commits:"
        head -5 /tmp/git_commits.txt | while IFS= read -r line; do
            print_color $YELLOW "  $line"
        done
    fi

    # Parse commits and group into sessions
    local current_author=""
    local current_session_start=""
    local current_session_end=""
    local total_seconds=0
    local session_count=0
    # Use files to store author hours for compatibility
    local author_hours_file="/tmp/author_hours.txt"
    > "$author_hours_file"

    while IFS='|' read -r timestamp author email hash subject; do
        # Skip if timestamp is empty
        if [[ -z "$timestamp" ]]; then
            continue
        fi

        # Convert timestamp to readable date
        local commit_date=$(date -r "$timestamp" '+%Y-%m-%d %H:%M:%S')

        # If this is the first commit or author changed, start new session
        if [[ -z "$current_session_start" ]] || [[ "$current_author" != "$author" ]]; then
            # If we had a previous session, calculate its duration
            if [[ -n "$current_session_start" ]] && [[ -n "$current_session_end" ]]; then
                local session_seconds=$((current_session_end - current_session_start))

                # Apply session constraints
                if [[ $session_seconds -gt $((MIN_SESSION_MINUTES * 60)) ]] && [[ $session_seconds -lt $((MAX_SESSION_HOURS * 3600)) ]]; then
                    total_seconds=$((total_seconds + session_seconds))
                    session_count=$((session_count + 1))
                    # Update author hours in file
                    local current_author_hours=$(grep "^$current_author|" "$author_hours_file" 2>/dev/null | cut -d'|' -f2 || echo "0")
                    local new_author_hours=$((current_author_hours + session_seconds))
                    grep -v "^$current_author|" "$author_hours_file" > "${author_hours_file}.tmp" 2>/dev/null || true
                    echo "$current_author|$new_author_hours" >> "${author_hours_file}.tmp"
                    mv "${author_hours_file}.tmp" "$author_hours_file"

                    if [[ "$verbose" == "true" ]]; then
                        local session_hours=$(seconds_to_hours $session_seconds)
                        print_color $YELLOW "  📝 Session $session_count: $current_author - ${session_hours}h"
                    fi
                fi
            fi

            # Start new session
            current_author="$author"
            current_session_start="$timestamp"
            current_session_end="$timestamp"
        else
            # Check if this commit is within the session gap
            local time_diff=$((timestamp - current_session_end))
            local gap_seconds=$((SESSION_GAP_HOURS * 3600))

            if [[ $time_diff -lt $gap_seconds ]]; then
                # Extend current session
                current_session_end="$timestamp"
            else
                # Close current session and start new one
                local session_seconds=$((current_session_end - current_session_start))

                if [[ $session_seconds -gt $((MIN_SESSION_MINUTES * 60)) ]] && [[ $session_seconds -lt $((MAX_SESSION_HOURS * 3600)) ]]; then
                    total_seconds=$((total_seconds + session_seconds))
                    session_count=$((session_count + 1))
                    # Update author hours in file
                    local current_author_hours=$(grep "^$current_author|" "$author_hours_file" 2>/dev/null | cut -d'|' -f2 || echo "0")
                    local new_author_hours=$((current_author_hours + session_seconds))
                    grep -v "^$current_author|" "$author_hours_file" > "${author_hours_file}.tmp" 2>/dev/null || true
                    echo "$current_author|$new_author_hours" >> "${author_hours_file}.tmp"
                    mv "${author_hours_file}.tmp" "$author_hours_file"

                    if [[ "$verbose" == "true" ]]; then
                        local session_hours=$(seconds_to_hours $session_seconds)
                        print_color $YELLOW "  📝 Session $session_count: $current_author - ${session_hours}h"
                    fi
                fi

                # Start new session
                current_session_start="$timestamp"
                current_session_end="$timestamp"
            fi
        fi
    done < /tmp/git_commits.txt

    # Process the last session
    if [[ -n "$current_session_start" ]] && [[ -n "$current_session_end" ]]; then
        local session_seconds=$((current_session_end - current_session_start))

        if [[ $session_seconds -gt $((MIN_SESSION_MINUTES * 60)) ]] && [[ $session_seconds -lt $((MAX_SESSION_HOURS * 3600)) ]]; then
            total_seconds=$((total_seconds + session_seconds))
            session_count=$((session_count + 1))
            # Update author hours in file
            local current_author_hours=$(grep "^$current_author|" "$author_hours_file" 2>/dev/null | cut -d'|' -f2 || echo "0")
            local new_author_hours=$((current_author_hours + session_seconds))
            grep -v "^$current_author|" "$author_hours_file" > "${author_hours_file}.tmp" 2>/dev/null || true
            echo "$current_author|$new_author_hours" >> "${author_hours_file}.tmp"
            mv "${author_hours_file}.tmp" "$author_hours_file"

            if [[ "$verbose" == "true" ]]; then
                local session_hours=$(seconds_to_hours $session_seconds)
                print_color $YELLOW "  📝 Session $session_count: $current_author - ${session_hours}h"
            fi
        fi
    fi

    # Clean up
    rm -f /tmp/git_commits.txt
    rm -f "$author_hours_file"

    # Calculate total hours
    local total_hours=$(seconds_to_hours $total_seconds)

    print_header "📈 Development Time Results"

    print_color $GREEN "✨ Total Development Time: ${total_hours} hours"
    print_color $BLUE "📊 Total Sessions: $session_count"
    print_color $BLUE "📝 Total Commits: $total_commits"

    echo
    print_color $PURPLE "👥 Per Author Breakdown:"
    if [[ -f "$author_hours_file" ]]; then
        while IFS='|' read -r author author_seconds; do
            if [[ -n "$author" ]] && [[ -n "$author_seconds" ]]; then
                local author_hours=$(seconds_to_hours $author_seconds)
                local percentage=$(echo "scale=1; $author_seconds * 100 / $total_seconds" | bc)
                print_color $YELLOW "  - $author: ${author_hours}h (${percentage}%)"
            fi
        done < "$author_hours_file"
    fi

    # Calculate recent activity (last 30 days)
    local thirty_days_ago=$(date -v-30d +%s 2>/dev/null || date -d "30 days ago" +%s)
    local recent_seconds=0

    git log --pretty=format:"%at|%an" --since="30 days ago" > /tmp/recent_commits.txt

    while IFS='|' read -r timestamp author; do
        if [[ -n "$timestamp" ]] && [[ "$timestamp" -gt "$thirty_days_ago" ]]; then
            recent_seconds=$((recent_seconds + 3600)) # Estimate 1 hour per commit for recent activity
        fi
    done < /tmp/recent_commits.txt

    rm -f /tmp/recent_commits.txt

    local recent_hours=$(seconds_to_hours $recent_seconds)

    echo
    print_color $PURPLE "📅 Recent Activity (Last 30 days):"
    print_color $YELLOW "  - Estimated: ${recent_hours} hours"

    # Generate badge URL
    local badge_url="https://img.shields.io/badge/development-${total_hours}_hours-blue"

    # Extract GitHub repo from origin URL
    local github_repo=""
    if git remote get-url origin &>/dev/null; then
        github_repo=$(git remote get-url origin | sed -E 's/.*github.com[\/:]([^/]+\/[^/.]+)(\.git)?.*/\1/')
    fi

    # If no GitHub repo found, use a default URL
    local badge_link="https://github.com/$github_repo"
    if [[ -z "$github_repo" ]]; then
        badge_link="https://github.com"
    fi

    echo
    print_color $CYAN "🛡️  Development Time Badge:"
    print_color $YELLOW "  Markdown: [![Development Time]($badge_url)]($badge_link)"

    echo
    print_color $GREEN "✅ Analysis complete!"
}

# Function to show help
show_help() {
    print_header "🕒 Development Time Estimation"
    echo "Usage: $0 [OPTIONS]"
    echo
    echo "Options:"
    echo "  -v, --verbose     Show detailed session information"
    echo "  -h, --help        Show this help message"
    echo
    echo "This script analyzes git commit history to estimate total development"
    echo "time based on commit timestamps and author information."
    echo
    echo "Estimation Method:"
    echo "  • Commits within $SESSION_GAP_HOURS hours are grouped into sessions"
    echo "  • Sessions shorter than $MIN_SESSION_MINUTES minutes are excluded"
    echo "  • Sessions longer than $MAX_SESSION_HOURS hours are capped"
    echo "  • Recent activity (last 30 days) estimated at 1 hour per commit"
    echo
}

# Parse command line arguments
verbose="false"

while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            verbose="true"
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            print_color $RED "❌ Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
    shift

done

# Check if bc is available for calculations
if ! command -v bc &> /dev/null; then
    print_color $RED "❌ 'bc' command not found. Please install it:"
    echo "  macOS: brew install bc"
    echo "  Ubuntu/Debian: sudo apt-get install bc"
    echo "  CentOS/RHEL: sudo yum install bc"
    exit 1
fi

# Run the analysis
calculate_development_time