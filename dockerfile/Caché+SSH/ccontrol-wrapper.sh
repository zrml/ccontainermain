#!/bin/bash

# Work around a werid overlayfs bug where files don't open properly if they haven't been
# touched first - see the yum-ovl plugin for a similar workaround
if [ "${1,,}" == "start" ]; then
    find / -name CACHE.DAT -exec touch {} \;
fi

/usr/local/etc/cachesys/ccontrol $@