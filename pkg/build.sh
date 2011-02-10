for i in web server oauth expvar websocket
do
    pkgdoc template.html github.com/garyburd/twister/$i > $i.html
done
