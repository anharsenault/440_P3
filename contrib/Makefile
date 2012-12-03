CC = go build

all: echoclient/echoclient echoserver/echoserver
#echostore/echostore

echoclient/echoclient:
	cd echoclient; $(CC) echoclient.go

echoserver/echoserver:
	cd echoserver; $(CC) echoserver.go

#echostore/echostore:
#	cd echostore; $(CC) echostore.go

.PHONY: clean kill test client

kill:
	ps aux | grep echoserver | awk '/atrejo/' | \
		awk '!/grep/ { print $$2 | "xargs kill -9" }'

drop:
	ps aux | grep echoserver | awk '/atrejo/' | \
		awk '!/grep/ { print $$2 | "xargs kill -9"; exit }'

#ps aux | grep echostore | awk '/atrejo/' | \
#		awk '!/grep/ { print $$2 | "xargs kill -9" }'

test:
	./echoserver/echoserver -i=1 -p=55455 :55456 :55457 :55458 :55459 &
	./echoserver/echoserver -i=2 -p=55456 :55455 :55457 :55458 :55459 &
	./echoserver/echoserver -i=3 -p=55457 :55455 :55456 :55458 :55459 &
	./echoserver/echoserver -i=4 -p=55458 :55455 :55456 :55457 :55459 &
	./echoserver/echoserver -i=5 -p=55459 :55455 :55456 :55457 :55458 &

client:
	./echoclient/echoclient -p=55458

clean: kill
	rm -rf echoclient/echoclient echoserver/echoserver
#rm echostore/echostore
