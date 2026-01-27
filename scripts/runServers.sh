#!/bin/bash

killall -9 client
killall -9 server

rm -rf ./var/log

bin/server 0&
bin/server 1&
bin/server 2&
bin/server 3&
bin/server 4&
bin/server 5&
bin/server 6&
bin/server 7&
bin/server 8&
bin/server 9&
bin/server 10&
bin/server 11&
bin/server 12&
bin/server 13&
bin/server 14&
bin/server 15&
bin/server 16&
bin/server 17&
bin/server 18&
bin/server 19&
bin/server 20&
bin/server 21&
bin/server 22&
bin/server 23&
bin/server 24&
bin/server 25&
bin/server 26&
bin/server 27&
bin/server 28&
bin/server 29&
bin/server 30&