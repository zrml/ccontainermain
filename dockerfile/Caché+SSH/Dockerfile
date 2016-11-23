# This Docker manifest file builds a container with:
# - sshd running (linux containers don't usually have it)
# - Caché 2016.2 and 
# - it handles container PID 1 via ccontainermain which offers various flags
#
# build the new image with:
# $ docker build --force-rm --no-cache -t cache:2016.2 .
#--

# pull from this repository
# note that if you don't have the distribution you're after it will be automatically
# downloaded from Docker central hub repository (you'll have to create a user there)
#
FROM tutum/centos:latest

MAINTAINER user <user@company.com>

# setup vars section___________________________________________________________________
#
ENV TMP_INSTALL_DIR=/tmp/distrib

# vars for Caché silent install
ENV ISC_PACKAGE_INSTANCENAME="CACHE"
ENV ISC_PACKAGE_INSTALLDIR="/usr/cachesys"
ENV ISC_PACKAGE_UNICODE="Y"

# Caché distribution file________________________________________________________________
# set-up and install Caché from distrib_tmp dir
RUN mkdir ${TMP_INSTALL_DIR}
WORKDIR ${TMP_INSTALL_DIR}

# update OS + dependencies & run Caché silent install___________________________________
RUN yum -y update && \
    yum -y install tar hostname net-tools which wget java && \

# Replace the following location with that of your Cache 2016.2 kit
    wget -O - 'https://replace_this_with_your_server/distrib/cache-2016.2.0.736.0-lnxrhx64.tar.gz' \

# Alternatively, if you're comfortable with all parties with access to to this docker image having
# access to these WRC credentials via the `docker history` command, comment out the above line,
# uncomment the following lines and fill in your WRC_USERNAME and WRC_PASSWORD to automatically
# fetch the kit from InterSystems' WRC.
 
#    WRC_USERNAME="user@company.com" && \
#    WRC_PASSWORD="your_password_here" && \
#      wget -qO /dev/null --keep-session-cookies --save-cookies /dev/stdout --post-data="UserName=$WRC_USERNAME&Password=$WRC_PASSWORD" 'https://login.intersystems.com/login/SSO.UI.Login.cls?referrer=https%253A//wrc.intersystems.com/wrc/login.csp' \
#        | wget -O - --load-cookies /dev/stdin 'https://wrc.intersystems.com/wrc/WRC.StreamServer.cls?FILE=/wrc/distrib/cache-2016.2.0.736.0-lnxrhx64.tar.gz' \

      | tar xvfzC - . && \
    ./cache-*/cinstall_silent && \
    rm -rf ${TMP_INSTALL_DIR}/* && \
    ccontrol stop $ISC_PACKAGE_INSTANCENAME quietly 
COPY cache.key $ISC_PACKAGE_INSTALLDIR/mgr/

# Workaround for an overlayfs bug which prevents Cache from starting with <PROTECT> errors
COPY ccontrol-wrapper.sh /usr/bin/
RUN cd /usr/bin                     && \
    rm ccontrol                     && \
    mv ccontrol-wrapper.sh ccontrol && \
    chmod 555 ccontrol

# TCP sockets that can be accessed if user wants to (see 'docker run -p' flag)
EXPOSE 57772 1972 22

# Caché container main process PID 1 (https://github.com/zrml/ccontainermain)
WORKDIR /
ADD ccontainermain .

ENTRYPOINT  ["/ccontainermain"]

# run via:
# docker run -d -p 57772:57772 -p 2222:22 -e ROOT_PASS="linux" <docker_image_id> -i=CACHE -xstart=/run.sh 
#
# more options & explinations
# $ docker run -d			// detached in the background; accessed only via network
# --privileged				// only for kernel =<3.16 like CentOS 6 & 7; it gives us root privileges to tune the kernel etc.
# -h <host_name>			// you can specify a host name
# -p 57772:57772 			// TCP socket port mapping as host_external:container_internal
# -p 0.0.0.0:2222:22		// this means allow 2222 to be accesses from any ip on this host and map it to port 22 in the container
# -e ROOT_PASS="linux"		// -e for env var; tutum/centos extension for root pwd definition
# <docker_image_id> 		// see docker images to fetch the right name & tag or id
# 							// after the Docker image id, we can specify all the flags supported by 'ccontainermain'
#							// see this page for more info https://github.com/zrml/ccontainermain
# -i=CACHE					// this is the Cachè instance name
# -xstart=/run.sh			// eXecute another service at startup time
#							// run.sh starts sshd (part of tutum centos container)
#							// for more info see https://docs.docker.com/reference/run/
#
