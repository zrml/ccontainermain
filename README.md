## ccontainermain

The program `ccontainermain` allows a Caché, Ensemble or HealthShare product to run in a Docker container.
Docker containers need a PID 1 or main process to hold up the container. This is what `ccontainermain` provides.
It is developed so that one can quickly work with Caché in a Docker container vs 
* having to understand why the container dies straight away and 
* having to develop a comprehensive script.

The name convention used is that of InterSystems commands found in the <installDir>/bin directory like:
ccontrol, cstart, cforce etc.
 
`ccontainermain` is called to run as the main process of a Docker container.
One would copy it in the container and specify it as the Dockerfile ENTRYPOINT argument, as the command to run. See Docker documentation on Dockerfile declarative manifesto.

`ccontainermain` start Caché|Enseble|HealthShare and logs any message and issues to the standard Docker logs output.
It also tries to tune shared memory so that Caché may start. You can pass higher value than the default 512MB that is usually enough to work.

`ccontainermain` also allows a software developer to start her or his Caché program and also other services.

However, the most important thing that ccontainermain does is probably the trapping of signals to the container.
Consider the Docker command:
 
	$ docker stop <running_container>

Docker gives a 10 seconds default and then bring the container down. Not ideal for a database using shared memory.
`ccontainermain` traps the signal and runs the Caché silent shutdown. Please remember to specify the -t (timeout) flag to the Docker stop command with a value >10 seconds as at times -depending on how busy the system is, it takes longer than that. Of course it all depends if one uses volumes or if one just uses a DB inside the container as an immutable artifact.

## options
`ccontainermain` offers several flags:
* -i for instance; it allows to specify the DB instance to start & stop; -i=CACHESYS2
* -n for namespace; it allows to specify the namespace where to run a program; -n=USER
* -r for routine; it allows to specify the routine name to start; -r=myApp or -r="##class(package.class).method()"
* -shmem for tuning SHMMAX; default val 512MB; -shmem=1024
* -xstart for eXecuting something else (example starting sshd etc.); -xstart=/usr/local/bin/runMyExtraServive.sh
* -xstop for eXectuing a stop of a started service; -xstop=/bringAllMyProcsDown.sh
* -cstart it's a boolean defaulted to true; It gives us the option to start a container without starting Caché; -cstart=true
* -cstop it's a boolean defaulted to true; it gives the option to skip the Caché shutdown; -cstop=false
* -nostu it's a boolean defaulted to false; it allows DB single user startup for maintenance mode 

The above flags can also be retrieved via

	$ ./ccontainermain -help


For more information on the Caché `ccontrol` related options please see:
[InteSystems documentation] (http://docs.intersystems.com/cache201511/csp/docbook/DocBook.UI.Page.cls?KEY=GSA_using_instance#GSA_using_instance_control)
and for the rest see
[the Docker documentation] (https://docs.docker.com/)

Please note that I've left in a debug (dbg) constant that you can use to get extra debugging information throughout the program.
For your convenience a linux executable has been provided so that you don't have to install GO and compile the code.
If you have GO installed simply

	$ go build ccontainermain.go


Please also note the dockerfile/ directory under which I'll try to upload few useful Dockerfile examples. Dockerfiles are Docker engine manifests that allows one to automate Docker images creation.

HTH


## TODO
* investigate SIGCHLD for dying processes and clean up PID table
* Windows/Azure support and testing

