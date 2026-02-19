import os

def create_grouped_bar_chart(filename, title, labels, baseline_vals, optimized_vals, ylabel, group_labels):
    max_val = max(max(baseline_vals), max(optimized_vals)) * 1.2
    svg_width = 600
    svg_height = 400
    margin_left = 80
    margin_bottom = 60
    margin_top = 60
    margin_right = 20
    
    chart_width = svg_width - margin_left - margin_right
    chart_height = svg_height - margin_top - margin_bottom
    
    group_width = chart_width / len(labels)
    bar_width = group_width * 0.35
    spacing = group_width * 0.05
    
    svg = f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {svg_width} {svg_height}" style="background-color: white; font-family: sans-serif;">
    <!-- Title -->
    <text x="{svg_width/2}" y="35" text-anchor="middle" font-size="20" font-weight="bold" fill="#333">{title}</text>
    
    <!-- Y Axis Label -->
    <text x="-{svg_height/2}" y="25" transform="rotate(-90)" text-anchor="middle" font-size="14" fill="#666">{ylabel}</text>
    
    <!-- Axes -->
    <line x1="{margin_left}" y1="{svg_height - margin_bottom}" x2="{svg_width - margin_right}" y2="{svg_height - margin_bottom}" stroke="#ccc" stroke-width="2"/>
    <line x1="{margin_left}" y1="{margin_top}" x2="{margin_left}" y2="{svg_height - margin_bottom}" stroke="#ccc" stroke-width="2"/>
    
    <!-- Legend -->
    <rect x="{svg_width - 200}" y="45" width="15" height="15" fill="#E24A33" rx="2" ry="2"/>
    <text x="{svg_width - 180}" y="57" font-size="12" fill="#333">{group_labels[0]}</text>
    <rect x="{svg_width - 90}" y="45" width="15" height="15" fill="#348ABD" rx="2" ry="2"/>
    <text x="{svg_width - 70}" y="57" font-size="12" fill="#333">{group_labels[1]}</text>
    '''

    for i, label in enumerate(labels):
        group_x = margin_left + (i * group_width)
        
        # Baseline Bar
        x1 = group_x + (group_width / 2) - bar_width - (spacing / 2)
        h1 = (baseline_vals[i] / max_val) * chart_height
        y1 = svg_height - margin_bottom - h1
        svg += f'<rect x="{x1}" y="{y1}" width="{bar_width}" height="{h1}" fill="#E24A33" rx="4" ry="4"/>\n'
        if baseline_vals[i] > 0:
            svg += f'<text x="{x1 + bar_width/2}" y="{y1 - 10}" text-anchor="middle" font-size="12" font-weight="bold" fill="#333">{baseline_vals[i]} MB</text>\n'
        
        # Optimized Bar
        x2 = group_x + (group_width / 2) + (spacing / 2)
        h2 = (optimized_vals[i] / max_val) * chart_height
        y2 = svg_height - margin_bottom - h2
        # For 0 MB, draw a tiny sliver just so it exists visually, but label it 0
        display_h2 = max(h2, 2) if optimized_vals[i] == 0 else h2
        display_y2 = svg_height - margin_bottom - display_h2
        svg += f'<rect x="{x2}" y="{display_y2}" width="{bar_width}" height="{display_h2}" fill="#348ABD" rx="4" ry="4"/>\n'
        svg += f'<text x="{x2 + bar_width/2}" y="{display_y2 - 10}" text-anchor="middle" font-size="12" font-weight="bold" fill="#333">{optimized_vals[i]} MB</text>\n'
        
        # Label Text
        svg += f'<text x="{group_x + group_width/2}" y="{svg_height - margin_bottom + 25}" text-anchor="middle" font-size="14" font-weight="bold" fill="#333">{label}</text>\n'

    svg += '</svg>'
    
    with open(filename, 'w') as f:
        f.write(svg)

def create_bar_chart(filename, title, labels, values, ylabel, colors):
    max_val = max(values) * 1.2
    svg_width = 600
    svg_height = 400
    margin_left = 80
    margin_bottom = 60
    margin_top = 60
    margin_right = 20
    
    chart_width = svg_width - margin_left - margin_right
    chart_height = svg_height - margin_top - margin_bottom
    
    bar_width = chart_width / (len(values) * 2)
    
    svg = f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {svg_width} {svg_height}" style="background-color: white; font-family: sans-serif;">
    <!-- Title -->
    <text x="{svg_width/2}" y="35" text-anchor="middle" font-size="20" font-weight="bold" fill="#333">{title}</text>
    
    <!-- Y Axis Label -->
    <text x="-{svg_height/2}" y="25" transform="rotate(-90)" text-anchor="middle" font-size="14" fill="#666">{ylabel}</text>
    
    <!-- Axes -->
    <line x1="{margin_left}" y1="{svg_height - margin_bottom}" x2="{svg_width - margin_right}" y2="{svg_height - margin_bottom}" stroke="#ccc" stroke-width="2"/>
    <line x1="{margin_left}" y1="{margin_top}" x2="{margin_left}" y2="{svg_height - margin_bottom}" stroke="#ccc" stroke-width="2"/>
    '''

    for i, (label, val, color) in enumerate(zip(labels, values, colors)):
        x = margin_left + (i * chart_width / len(values)) + (chart_width / len(values) - bar_width) / 2
        bar_h = (val / max_val) * chart_height
        y = svg_height - margin_bottom - bar_h
        
        # Bar
        svg += f'<rect x="{x}" y="{y}" width="{bar_width}" height="{bar_h}" fill="{color}" rx="4" ry="4"/>\n'
        
        # Value Text
        svg += f'<text x="{x + bar_width/2}" y="{y - 10}" text-anchor="middle" font-size="14" font-weight="bold" fill="#333">{val} MB</text>\n'
        
        # Label Text
        svg += f'<text x="{x + bar_width/2}" y="{svg_height - margin_bottom + 25}" text-anchor="middle" font-size="14" font-weight="bold" fill="#333">{label}</text>\n'

    svg += '</svg>'
    
    with open(filename, 'w') as f:
        f.write(svg)


# Chart 1: Kind Cluster Profiling (Grouped Bar Chart showing Total Heap vs Specific Allocation)
create_grouped_bar_chart(
    'research/repro/report/heap_comparison.svg',
    'Live Cluster Heap Breakdown (2,000 ConfigMaps)',
    ['Total Heap (inuse_space)', 'FieldsV1.Unmarshal Allocations'],
    [146.5, 5.0],  # Baseline
    [141.7, 0.0],  # Optimized
    'Heap Memory (MB)',
    ['Baseline ([]byte)', 'Optimized (String)']
)

# Chart 2: Pine Megacluster Simulation (Retained Memory for 200k objects)
create_bar_chart(
    'research/repro/report/pine_simulation.svg',
    'Pine Simulation: Retained Memory (200,000 Replicas)',
    ['Baseline ([]byte)', 'Optimized (String Interning)'],
    [61.2, 4.7],
    'Retained Heap Memory (MB)',
    ['#E24A33', '#348ABD']
)

print("SVGs generated successfully.")
