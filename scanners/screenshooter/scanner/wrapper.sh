# Screnshooter Entrypoint Script to change the result file linux permission after completion. Chrome will set the permission in a way which makes it inaccassible to the lurcher otherwise
/gowitness $@
chmod a=r /home/securecodebox/screenshot.png
exit 0