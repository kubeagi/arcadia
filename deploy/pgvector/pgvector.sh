#!/bin/bash
set -e
# install git
apt-get update
apt-get install -y git
# clone pgvector
git clone --branch v0.5.1 https://github.com/pgvector/pgvector.git /tmp/pgvector
# install pg packages for make install pgvector
apt install -y postgresql-common gnupg2
export YES=yes && /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh
apt-get update
apt-mark hold locales
apt-get install -y --no-install-recommends build-essential postgresql-server-dev-16
# build pgvector
cd /tmp/pgvector
make clean
make OPTFLAGS=""
make install
mkdir /usr/share/doc/pgvector
cp LICENSE README.md /usr/share/doc/pgvector
rm -r /tmp/pgvector
apt-get remove -y build-essential postgresql-server-dev-16
apt-get autoremove -y
apt-mark unhold locales
rm -rf /var/lib/apt/lists/*
