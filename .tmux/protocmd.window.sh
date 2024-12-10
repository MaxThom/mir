# Set window root path. Default is `$session_root`.
# Must be called before `new_window`.
window_root "$PWD/cmds/protocmd/"

# Create new window. If no argument is given, window name will be based on
# layout file name.
new_window "protocmd"

#run_cmd "sleep 5 && go run main.go"
run_cmd 'bash -c "(sleep 5 && cd ../../ && air -c .air/protocmd.toml)"'


# Split window into panes.
split_v 20
#split_h 50

# Run commands.
#run_cmd "top"     # runs in active pane
run_cmd "cd ../../internal/servers/protocmd_srv/ && clear"
send_keys "go test -run TestPublishCmdRequest"

# Paste text
#send_keys "top"    # paste into active pane
#send_keys "date" 1 # paste into pane 1

# Set active pane.
#select_pane 0
