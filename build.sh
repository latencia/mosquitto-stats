#!/bin/bash
set -e

NAME=mqtt-stats
VERSION=0.1

BUILD_DIR=$HOME/debian/tmp/$NAME
PKG_NAME=$NAME-$VERSION
PKG=$BUILD_DIR/$PKG_NAME.tar.gz
DEB_TARGET_DIR=$HOME/debian/$NAME
ORIG_TARBALL=$DEB_TARGET_DIR/${NAME}_$VERSION.orig.tar.gz

orig_tarball() {
  rm -rf $BUILD_DIR && mkdir -p $BUILD_DIR
	git archive --output=$PKG --prefix=$PKG_NAME/ HEAD
	mv $PKG $BUILD_DIR/${NAME}_$VERSION.orig.tar.gz
}

srcdeb() {
  orig_tarball

	mkdir -p $DEB_TARGET_DIR
	# Prevents overwriting a good tarball
  if [ -f "$ORIG_TARBALL" ] && [ "$ARG1" != "--force" ]; then
    echo "Original tarball previously created, use --force to overwrite"
    exit 1
  fi

	cd $BUILD_DIR && \
		tar xzf ${NAME}_$VERSION.orig.tar.gz && \
	  cd $NAME-$VERSION && \
		debuild -S && rm -rf $BUILD_DIR/$NAME-$VERSION && \
		mv $BUILD_DIR/* $DEB_TARGET_DIR
}

CMD=$1
ARG1=$2
case $CMD in
  srcdeb)
    srcdeb
    ;;
  deps)
    rm -rf vendor/src
    GOBIN=$PWD/vendor/bin GOPATH=$PWD/vendor go get
    rm -rf vendor/pkg
    rm -rf vendor/bin
    find vendor/src -type d -name '.git' -o -name '.hg' | xargs rm -rf
    find vendor/src -type f -name '.gitignore' -o -name '.hgignore' | xargs rm
    ;;
  format)
    gofmt -s -w **/*.go
    ;;
  *)
	  GOPATH=$PWD/vendor go build -o $NAME
    ;;
esac