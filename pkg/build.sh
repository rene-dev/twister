for i in web server oauth expvar websocket
do
    pkgdoc -basePath=http://github.com/garyburd/twister/blob/master/$i/ template.html github.com/garyburd/twister/$i > $i.html
done
