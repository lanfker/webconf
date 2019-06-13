# CVDP LAN device configuration portal

## Enhanced capability

The previous *SimpleHTTPServer.py* only handles file upload and download. It does nothing specifically tailored for the CVDP project. 

This project brings the following features to ordinary users: 

* Ability to edit json content, then save. *The is no `save as` option yet*
* Ability to activate different onboard configuration files
* Ability to reboot the device to make changes effective
* Ability to specify SSID and password 
* Of course, file upload and download

## Visit

    http://192.168.46.30   (no port required)

## Q&A

* What if I want to change an SSID's password? 

    Just specify a connection with the same SSID, the new information will overwrite the old information on board.

* Why do we need to restart the CVDP to makes thing effective

    The CVDP software that collects CAN data only parse config.json at the very beginning. Changes after the CVDP binary has been running takes no effect. 

    Besides, revisions on WIFI connections also require CVDP to be rebooted.

    **Technically, we do not have to reboot the device to make things effective, but that involves more work**. I wish to keep things simple.


## TODO

* The Golang code is ugly, I have not thought about refactor the code. Will do shortly
* Add comments to the Golang code
