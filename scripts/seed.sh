#!/bin/bash
# TODO create on multiline


bin="./bin/mir"

echo "DELETED"
$bin device delete --target.labels "source=seed" -o json

$bin device create --random-id --name "clean_hvac"  --namespace "monitoring" --desc "To monitor and control hvac system"      --labels "source=seed;factory=A;room=clean"     --anno "utility=low;managed=true"       >/dev/null
$bin device create --random-id --name "server_temp" --namespace "monitoring" --desc "To monitor and control temperature"      --labels "source=seed;factory=A;room=server"    --anno "utility=moderate;managed=true"  >/dev/null
$bin device create --random-id --name "server_env"  --namespace "monitoring" --desc "To monitor environmental data"           --labels "source=seed;factory=B;room=server"    --anno "utility=critical;managed=true"  >/dev/null
$bin device create --random-id --name "bob_farts"   --namespace "monitoring" --desc "To monitor bob's car seat fart level"    --labels "source=seed;factory=A;room=car"       --anno "utility=critical;managed=false" >/dev/null
$bin device create --random-id --name "spod_guard"  --namespace "monitoring" --desc "To take control of spod the dog robot"   --labels "source=seed;factory=B;room=hallway"   --anno "utility=critical;managed=true"  >/dev/null

$bin device create --random-id --name "vroom_vroom" --namespace "car" --desc "Mazda3 Sport autocar"        --labels "source=seed;fleet=mtrl;maker=mazda"   --anno "last_inspection=never;ac=enabled"                  >/dev/null
$bin device create --random-id --name "laval_rep"   --namespace "car" --desc "Honda Civir autocar"         --labels "source=seed;fleet=mtrl;maker=honda"   --anno "last_inspection=today;ac=disabled"     --disabled  >/dev/null
$bin device create --random-id --name "thomthom"    --namespace "car" --desc "Toyota Aqua autocar"         --labels "source=seed;fleet=mtrl;maker=toyota"  --anno "last_inspection=never;ac=enabled"                  >/dev/null
$bin device create --random-id --name "BC_neck"     --namespace "car" --desc "F150 autocar"                --labels "source=seed;fleet=paris;maker=ford"   --anno "last_inspection=yesterday;ac=enabled"  --disabled  >/dev/null
$bin device create --random-id --name "evil_within" --namespace "car" --desc "Google Autonomous autocar"   --labels "source=seed;fleet=paris;maker=google" --anno "last_inspection=tomorrow;ac=disabled"              >/dev/null

$bin device create --random-id --name "hubble"      --namespace "satellite" --desc "Hubble"      --labels "source=seed;location=LEO;role=pic"        --anno "deorbit=2034;"  >/dev/null
$bin device create --random-id --name "james_webb"  --namespace "satellite" --desc "James Webb"  --labels "source=seed;location=lagrange;role=pic"   --anno "deorbit=2065;"  >/dev/null
$bin device create --random-id --name "chandra"     --namespace "satellite" --desc "Chandra"     --labels "source=seed;location=LEO;role=x-ray"      --anno "deorbit=2100;"  >/dev/null
$bin device create --random-id --name "fermi"       --namespace "satellite" --desc "Fermi"       --labels "source=seed;location=LEO;role=gamma-ray"  --anno "deorbit=1992;"  >/dev/null
$bin device create --random-id --name "nustar"      --namespace "satellite" --desc "NuSTAR"      --labels "source=seed;location=LEO;role=nuclear"    --anno "deorbit=2014;"  >/dev/null

$bin device create --random-id --name "l_one"   --namespace "mushrooms" --desc "Mushroom lamp one"   --labels "source=seed;location=ile_du_vent;version=1"   --anno "color=rainbow;made_by=seb" --disabled  >/dev/null
$bin device create --random-id --name "l_two"   --namespace "mushrooms" --desc "Mushroom lamp two"   --labels "source=seed;location=boreal;version=2"        --anno "color=blue;made_by=max"                >/dev/null
$bin device create --random-id --name "l_tree"  --namespace "mushrooms" --desc "Mushroom lamp three" --labels "source=seed;location=boreal;version=3"        --anno "color=green;made_by=max"               >/dev/null
$bin device create --random-id --name "l_four"  --namespace "mushrooms" --desc "Mushroom lamp four"  --labels "source=seed;location=shipwreck;version=2"     --anno "color=purple;made_by=seb"  --disabled  >/dev/null
$bin device create --random-id --name "l_five"  --namespace "mushrooms" --desc "Mushroom lamp five"  --labels "source=seed;location=shipwreck;version=3"     --anno "color=cyan;made_by=liam"               >/dev/null

$bin device create --id "0xf86tlm" --name "dev1" --namespace "dev" --desc "Device hard one" --labels "source=seed;" --anno "" >/dev/null
$bin device create --id "0xf86cmd" --name "dev2" --namespace "dev" --desc "Device hard two" --labels "source=seed;" --anno "" >/dev/null

echo "CREATED"
$bin device list --target.labels "source=seed" -o json
