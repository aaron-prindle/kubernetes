set terminal pngcairo size 800,600 enhanced font 'Arial,12'
set output 'docs/memory_reduction_plot.png'

set title 'Memory Footprint at 50,000 Pods: master vs experimental' font ',14'
set ylabel 'Memory Allocated (MB)'
set style data histograms
set style histogram cluster gap 1
set style fill solid 0.8 border -1
set boxwidth 0.9

set yrange [0:1600]
set grid ytics

$data << EOD
Branch          Total_Heap   FieldsV1
"master"        1450         130.59
"experimental"  1370         27.52
EOD

plot $data using 2:xtic(1) title "Total Apiserver Heap" lc rgb "steelblue", $data using 3 title "FieldsV1 Allocation" lc rgb "orange", $data using ($0-0.15):2:(sprintf("%g MB", $2)) with labels offset 0,0.7 title "", $data using ($0+0.15):3:(sprintf("%g MB", $3)) with labels offset 0,0.7 title ""
