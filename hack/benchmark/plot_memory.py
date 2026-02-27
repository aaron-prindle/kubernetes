import matplotlib.pyplot as plt
import numpy as np

# Data points collected from our benchmark runs
pods = np.array([50, 5000, 50000])
baseline_mb = np.array([0, 10.01, 130.59]) # master
interned_mb = np.array([0, 2.50, 27.52])   # experimental branch

# Create a figure and axis
plt.figure(figsize=(10, 6))

# Plot the lines
plt.plot(pods, baseline_mb, marker='o', linestyle='-', color='red', linewidth=2, label='master (Baseline []byte)')
plt.plot(pods, interned_mb, marker='s', linestyle='-', color='green', linewidth=2, label='experimental (stringhandle)')

# Fill the area between the lines to show the memory savings
plt.fill_between(pods, baseline_mb, interned_mb, color='lightgreen', alpha=0.3, label='Memory Saved')

# Add labels and title
plt.title('API Server WatchCache Memory: metav1.FieldsV1 Allocations', fontsize=14, pad=15)
plt.xlabel('Number of Duplicated Pods (Running)', fontsize=12)
plt.ylabel('Memory Allocated (MB)', fontsize=12)

# Set grid and legend
plt.grid(True, linestyle='--', alpha=0.7)
plt.legend(fontsize=11, loc='upper left')

# Customize x-axis to show the scale better
plt.xscale('linear')
plt.xticks(np.arange(0, 50001, 10000))

# Add data labels to the points for clarity
for i, txt in enumerate(baseline_mb):
    if txt > 0:
        plt.annotate(f"{txt:.2f} MB", (pods[i], baseline_mb[i]), textcoords="offset points", xytext=(0,10), ha='center', fontsize=9, color='darkred')
        
for i, txt in enumerate(interned_mb):
    if txt > 0:
        plt.annotate(f"{txt:.2f} MB", (pods[i], interned_mb[i]), textcoords="offset points", xytext=(0,-15), ha='center', fontsize=9, color='darkgreen')

# Save the plot
plt.tight_layout()
plt.savefig('docs/memory_scaling_plot.png', dpi=300)
print("Plot saved to docs/memory_scaling_plot.png")
