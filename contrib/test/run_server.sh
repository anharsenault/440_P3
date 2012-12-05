#!/bin/bash

./echoserver/echoserver -i=1 -p=55455 :55456 :55457 :55458 :55459 &
./echoserver/echoserver -i=2 -p=55456 :55455 :55457 :55458 :55459 &
./echoserver/echoserver -i=3 -p=55457 :55455 :55456 :55458 :55459 &
./echoserver/echoserver -i=4 -p=55458 :55455 :55456 :55457 :55459 &
./echoserver/echoserver -i=5 -p=55459 :55455 :55456 :55457 :55458 &
