#!/usr/bin/python
# -*- coding: UTF-8 -*-

import sys
import urllib
import bencode
import hashlib
import base64

if len(sys.argv) == 0:
    print("Usage: file")
    exit()

torrent = open(sys.argv[1], 'r').read()
metadata = bencode.bdecode(torrent)

hashcontents = bencode.bencode(metadata['info'])
hash = hashlib.sha1(hashcontents).hexdigest()
name = metadata['info']['name']

magneturi = 'magnet:?xt=urn:btih:' + hash + '&dn=' + name 
print(magneturi) 
