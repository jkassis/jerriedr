# jerriedr
FROM golang:1.18-alpine

# Put the maintainer name in the image metadata
LABEL maintainer="Jeremy Kassis <jkassis@gmail.com>"

# Rename the builder environment variable to inform users about application you provide them
ENV BUILDER_VERSION 1.0

# Set labels used in OpenShift to describe the builder image
LABEL io.k8s.description="Platform for building jerriedr" \
     io.k8s.display-name="jerriedr Builder 1.0.0" \
     io.openshift.expose-services="10000:http" \
     io.openshift.tags="builder,x.y.z,jerriedr"


# # ROCKSDB
RUN echo "@testing http://nl.alpinelinux.org/alpine/edge/testing" >>/etc/apk/repositories
RUN echo "@testing http://nl.alpinelinux.org/alpine/edge/main" >>/etc/apk/repositories
RUN apk add --update --no-cache build-base linux-headers git cmake bash rsync #wget mercurial g++ autoconf libgflags-dev cmake bash perl
RUN apk add --update --no-cache zlib zlib-dev bzip2 bzip2-dev snappy snappy-dev lz4 lz4-dev zstd@testing zstd-dev@testing libtbb-dev@testing libtbb@testing


# # installing latest gflags
# RUN cd /tmp && \
#     git clone https://github.com/gflags/gflags.git && \
#     cd gflags && \
#     mkdir build && \
#     cd build && \
#     cmake -DBUILD_SHARED_LIBS=1 -DGFLAGS_INSTALL_SHARED_LIBS=1 .. && \
#     make install && \
#     cd /tmp && \
#     rm -R /tmp/gflags/

# # Install Rocksdb
# RUN cd /tmp && \
#     git clone https://github.com/facebook/rocksdb.git && \
#     cd rocksdb && \
#     git checkout v6.14.6 && \
#     make shared_lib && \
#     mkdir -p /usr/local/rocksdb/lib && \
#     mkdir /usr/local/rocksdb/include && \
#     cp librocksdb.so* /usr/local/rocksdb/lib && \
#     cp /usr/local/rocksdb/lib/librocksdb.so* /usr/lib/ && \
#     cp -r include /usr/local/rocksdb/ && \
#     cp -r include/* /usr/include/ && \
#     rm -R /tmp/rocksdb/

# # For cgo
# RUN apk add --update gcc g++ glib-dev
# ENV CGO_CFLAGS="-I/usr/local/rocksdb/include"
# ENV CGO_LDFLAGS="-L/usr/local/rocksdb/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy"
# ENV CGO_CFLAGS_ALLOW='.*'
# ENV CGO_CPPFLAGS="-Wno-error -Wno-nullability-completeness -Wno-expansion-to-defined -Wno-builtin-requires-header"

# Copy the S2I scripts to /usr/libexec/s2i, since openshift/base-centos7 image
# sets io.openshift.s2i.scripts-url label that way, or update that label
LABEL io.openshift.s2i.scripts-url=image:///usr/libexec/s2i
COPY ./s2i/bin/ /usr/libexec/s2i

# cache
RUN mkdir /.cache
RUN chown -R 1001:1001 /.cache

# standard stuff
WORKDIR /opt/app-root
RUN adduser --disabled-password --gecos '' --uid 1001 adam
RUN chown -R 1001:1001 /opt/app-root
USER 1001
EXPOSE 10000
CMD ["/usr/libexec/s2i/usage"]
