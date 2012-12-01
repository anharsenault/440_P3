CC = go build

all: echoclient/echoclient echoserver/echoserver echostore/echostore

echoclient/echoclient:
	cd echoclient; $(CC) echoclient.go

echoserver/echoserver:
	cd echoserver; $(CC) echoserver.go

echostore/echostore:
	cd echostore; $(CC) echostore.go

.PHONY: clean kill test client

clean:
	rm echoclient/echoclient; rm echoserver/echoserver; rm echostore/echostore

kill:
	ps aux | grep echoserver | awk '/atrejo/' | \
		awk '!/grep/ { print $$2 | "xargs kill -9" }'; \
	ps aux | grep echostore | awk '/atrejo/' | \
		awk '!/grep/ { print $$2 | "xargs kill -9" }'

test:
	./echoserver/echoserver -p=55455 &

client:
	./echoclient/echoclient -p=55455
