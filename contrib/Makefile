CC = go build

all: echoclient/echoclient echoserver/echoserver echostore/echostore

echoclient/echoclient:
	cd echoclient; $(CC) echoclient.go

echoserver/echoserver:
	cd echoserver; $(CC) echoserver.go

echostore/echostore:
	cd echostore; $(CC) echostore.go

.PHONY: clean kill test

kill:
	./test/kill_all.sh

clean: kill
	rm -rf echoclient/echoclient echoserver/echoserver echostore/echostore
