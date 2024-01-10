#!/bin/bash
set -e

# 0. remove old files
find . | grep -v run.sh | grep -v README.md | grep -v '^\.$' | xargs rm -r -f

# 1. get base dockerfile and script from bitnami
git clone -n --depth=1 --filter=tree:0 https://github.com/bitnami/containers.git
cd containers
git sparse-checkout set --no-cone bitnami/postgresql/16/debian-11
git checkout
mv bitnami/postgresql/16/debian-11/* ..
cd ..
rm -r -f containers

# 2. add pgvector build script
cat >pgvector.sh <<EOF
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
EOF

chmod +x pgvector.sh

sed -i'' -e '/EXPOSE 5432/a\
COPY pgvector.sh .\
RUN bash -x pgvector.sh' Dockerfile

rm -f Dockerfile-e
