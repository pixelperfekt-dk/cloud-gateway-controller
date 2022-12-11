#!/bin/sh

make build
HASH=`ls -lR pkg/ cmd/ | egrep -v '*.~' | md5sum`
while true; do
    NEWHASH=`ls -lR pkg/ cmd/ | egrep -v '*.~' | md5sum`
    if [ "$HASH" != "$NEWHASH" ]; then
	HASH="$NEWHASH"
	if make build; then
	    echo "*** Build OK!"
	fi
    fi
    sleep 2
done
