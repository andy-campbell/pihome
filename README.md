pihome
======

Web app to run on a raspberry pi which starts my main server. Allows for some power saving to occur....

======


A Small home web server that I run on my raspberry pi. 

It currently has the ability to log in / out and turn another
device on and off. As well a be able to determine if the remote device
is currently active.

- Turning on is done with a wol packet sent to the mac.
- Turning off is done with client.go listening for a connection on a specific port
if it sees a connection occur then it will turn off the remote device with turn off.
- Checking if the remote device is active is done by trying to open port 22 (ssh) on
the remote device. If that succeeds then the remote device is concidered active.


To add your own configuration do the following:
- touch config.xml
- Open config.xml with editor of choice.
Add the following and change as you see fit

<Config>
	<ServAddr>Dragon:22</ServAddr>
	<MacAddr>00:1e:c9:2d:d6:d9</MacAddr>
	<UserName>andrew</UserName>
	<Password>*******</Password>
	<FullName>Andrew Campbell</FullName>
</Config>

To build run the following command:
- go build home.go


