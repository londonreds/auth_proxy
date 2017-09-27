# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure(2) do |config|
  config.vm.box = "ubuntu/trusty64"
  config.vm.synced_folder ".", "/opt/go/src/github.com/buzzfeed/auth_proxy"
  config.vm.network "private_network", ip: "10.0.0.109"
  config.vm.provision "shell", inline: <<-SHELL
    set -e

    apt-get update
    apt-get install -y git
    chown -R vagrant:vagrant /opt/go

    cd /tmp
    if [ ! -f /usr/local/bin/go ]; then
        wget --quiet https://storage.googleapis.com/golang/go1.4.2.linux-amd64.tar.gz -O go-1.4.2.tar.gz
        tar -C /usr/local -xzf go-1.4.2.tar.gz
        ln -s /usr/local/go/bin/* /usr/local/bin
    fi

    if [ ! -f /usr/local/bin/gpm ]; then
        wget --quiet https://github.com/pote/gpm/archive/v1.4.0.tar.gz -O gpm-1.4.0.tar.gz
        tar -C /usr/local -xzf gpm-1.4.0.tar.gz
        mv /usr/local/gpm-1.4.0 /usr/local/gpm
        ln -s /usr/local/gpm/bin/gpm /usr/local/bin
    fi

    mkdir -p /opt/go/src/github.com/buzzfeed/auth_proxy/.godeps
    chown -R vagrant:vagrant /opt/go/src/github.com/buzzfeed/auth_proxy/.godeps
    echo "export GOPATH=/opt/go/src/github.com/buzzfeed/auth_proxy/.godeps:/opt/go" > /etc/profile.d/go.sh
    echo "export PATH=/opt/go/bin:/opt/go/src/github.com/buzzfeed/auth_proxy/.godeps/bin:$PATH" >> /etc/profile.d/go.sh

    if ! grep "cd /opt/go" /home/vagrant/.profile ; then
        echo "cd /opt/go/src/github.com/buzzfeed/auth_proxy" >> /home/vagrant/.profile
    fi

  SHELL
end
