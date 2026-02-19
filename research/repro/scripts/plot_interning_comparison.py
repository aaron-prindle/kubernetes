import matplotlib.pyplot as plt

# Data
labels = ['Baseline ([]byte)', 'Optimized (String + Interning)']
memory_mb = [61.17, 4.72]

fig, ax = plt.subplots(figsize=(8, 6))
bars = ax.bar(labels, memory_mb, color=['#E24A33', '#348ABD'])
ax.set_ylabel('Retained Heap Memory (MB)', fontsize=12)
ax.set_title('Cache Memory for 200,000 Identical FieldsV1 Entries', fontsize=14, pad=20)
ax.set_ylim(0, 75)

# Add data labels
for bar in bars:
    height = bar.get_height()
    ax.text(bar.get_x() + bar.get_width()/2., height + 1.5,
            f'{height:.2f} MB',
            ha='center', va='bottom', fontweight='bold', fontsize=12)

# Subtitle explaining the composition
ax.text(0.5, -0.15, 'Simulates apiserver watch cache storing duplicated replica metadata.\nOptimized version only retains slice headers (4.72 MB), deduplicating all payload data.',
        ha='center', va='top', transform=ax.transAxes, fontsize=10, style='italic', color='gray')

plt.tight_layout()
plt.savefig('/usr/local/google/home/aprindle/kubernetes/research/repro/report/09_interning_retained_memory.png', dpi=300)
