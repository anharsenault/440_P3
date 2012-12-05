CC = go build

all: echoclient/echoclient echoserver/echoserver echostore/echostore cmdlineclient/cmdlineclient httpserver/httpserver

echoclient/echoclient:
	cd echoclient; $(CC) echoclient.go

cmdlineclient/cmdlineclient:
	cd cmdlineclient; $(CC) cmdlineclient.go

echoserver/echoserver:
	cd echoserver; $(CC) echoserver.go

echostore/echostore:
	cd echostore; $(CC) echostore.go

httpserver/httpserver:
	cd httpserver; $(CC) httpserver.go

.PHONY: clean kill test

kill:
	./test/kill_all.sh

clean: kill
	rm -rf echoclient/echoclient echoserver/echoserver echostore/echostore cmdlineclient/cmdlineclient httpserver/httpserver
