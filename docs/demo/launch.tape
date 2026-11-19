# Hidden launch shared by most demo tapes: a fresh scratch data dir seeded by
# scripts/demoseed, and the app run with an empty PATH so it finds no audio player or
# notifier (recording never plays sounds or posts notifications on this machine).
Hide
Type `export PS1='$ ' D=$(mktemp -d) && ./bin/demoseed --data-dir "$D" >/dev/null && clear`
Enter
Sleep 2s
Type `PATH=/var/empty ./bin/unitock --data-dir "$D"`
Enter
Wait+Screen /Start task/
Sleep 300ms
Show
