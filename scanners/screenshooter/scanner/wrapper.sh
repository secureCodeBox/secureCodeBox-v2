# Screnshooter Entrypoint Script to change the result file linux permission after completion. Firefox will set the permission in a way which makes it inaccassible to the lurcher otherwise
firefox $@
chmod a=r /home/securecodebox/screenshot.png
exit 0