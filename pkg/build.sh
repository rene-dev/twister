for i in web server oauth expvar websocket
do
    pkgdoc -basePath=http://github.com/garyburd/twister/blob/`(cd $GOROOT/src/pkg/github.com/garyburd/twister/; git rev-parse HEAD)`/$i/ template.html github.com/garyburd/twister/$i > $i.html
done
