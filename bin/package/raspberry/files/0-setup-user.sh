#!/bin/bash -ev

usermod -l myst pi
usermod -m -d /home/myst myst
echo 'myst:mystberry'|chpasswd

rm /etc/sudoers.d/010_pi-nopasswd
install -m 644 myst_sudo_nopasswd /etc/sudoers.d/010_myst-nopasswd
