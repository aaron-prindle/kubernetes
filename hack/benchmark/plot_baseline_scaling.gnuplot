set terminal pngcairo size 800,600 enhanced font 'Arial,12'
set output 'docs/baseline_scaling_plot.png'

set title 'Baseline (master) Memory Bloat: metav1.FieldsV1 Allocations' font ',14'
set xlabel 'Number of Duplicated Pods (Running)'
set ylabel 'Memory Allocated (MB)'

set grid
set key top left
set xrange [-1000:55000]
set yrange [0:150]

# Add labels to points
set label "16.0 MB" at 1000, 22 center textcolor rgb "red"
set label "41.5 MB" at 10000, 48 center textcolor rgb "red"
set label "134.6 MB" at 50000, 142 center textcolor rgb "red"

# Define data points inline
$data << EOD
# Pods  Master_MB
1000    16.02
10000   41.54
50000   134.60
EOD

plot $data using 1:2 with linespoints lw 3 pt 7 ps 1.5 lc rgb "red" title 'master (Baseline []byte)'
