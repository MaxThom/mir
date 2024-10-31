# Set a custom session root path. Default is `$HOME`.
# Must be called before `initialize_session`.
session_root "$PWD"

# Create session with specified name if it does not already exist. If no
# argument is given, session name will be based on layout file name.
if initialize_session "mir"; then

	# Create a new window inline within session layout definition.
	#new_window "misc"

	# Load a defined window layout.
	load_window "./.tmux/mir.window.sh"
	load_window "./.tmux/wiki.window.sh"
	load_window "./.tmux/cfg.window.sh"
	load_window "./.tmux/core.window.sh"
	load_window "./.tmux/prototlm.window.sh"
	load_window "./.tmux/protocmd.window.sh"
	load_window "./.tmux/nats.window.sh"
	load_window "./.tmux/surreal.window.sh"
	load_window "./.tmux/influxdb.window.sh"
	load_window "./.tmux/promstack.window.sh"
	load_window "./.tmux/book.window.sh"

	# Select the default active window on session creation.
	select_window 1

fi

# Finalize session creation and switch/attach to it.
finalize_and_go_to_session
