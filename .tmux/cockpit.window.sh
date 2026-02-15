# Set window root path. Default is `$session_root`.
# Must be called before `new_window`.
window_root "$PWD/cmds/cockpit/"

# Create new window. If no argument is given, window name will be based on
# layout file name.
new_window "cockpit"

#run_cmd "sleep 5 && go run main.go"
run_cmd 'bash -c "(sleep 5 && cd ../../ && air -c .air/cockpit.toml)"'


# Split window into panes.
split_v 50
#split_h 50

# Run commands.
#run_cmd "top"     # runs in active pane
#run_cmd "date" 1  # runs in pane 1
run_cmd 'bash -c "(sleep 5 && cd ../../internal/ui/web && npm run dev)"'

# Paste text
#send_keys "top"    # paste into active pane
#send_keys "date" 1 # paste into pane 1

# Set active pane.
#select_pane 0
