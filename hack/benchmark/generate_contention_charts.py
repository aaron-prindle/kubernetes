import math

def create_bar_chart(filename, title, labels, baseline_vals, optimized_vals, ylabel):
    svg_width = 800
    svg_height = 450
    margin_left = 100
    margin_bottom = 60
    margin_top = 80
    margin_right = 40
    
    chart_width = svg_width - margin_left - margin_right
    chart_height = svg_height - margin_top - margin_bottom
    
    max_val = max(max(baseline_vals), max(optimized_vals)) * 1.2
    
    svg = f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {svg_width} {svg_height}" style="background-color: white; font-family: sans-serif;">
    <text x="{svg_width/2}" y="40" text-anchor="middle" font-size="20" font-weight="bold" fill="#333">{title}</text>
    <text x="-{svg_height/2}" y="30" transform="rotate(-90)" text-anchor="middle" font-size="14" fill="#666">{ylabel}</text>
    
    <rect x="{svg_width - 250}" y="20" width="15" height="15" fill="#E24A33"/>
    <text x="{svg_width - 230}" y="32" font-size="12" fill="#333">master ([]byte)</text>
    <rect x="{svg_width - 250}" y="45" width="15" height="15" fill="#348ABD"/>
    <text x="{svg_width - 230}" y="57" font-size="12" fill="#333">experimental (interned)</text>
    '''

    for i, label in enumerate(labels):
        x_base = margin_left + (i * (chart_width / len(labels))) + (chart_width / (len(labels) * 4))
        bar_width = chart_width / (len(labels) * 3)
        
        # Master Bar
        h_b = (baseline_vals[i] / max_val) * chart_height
        svg += f'<rect x="{x_base}" y="{svg_height - margin_bottom - h_b}" width="{bar_width}" height="{h_b}" fill="#E24A33"/>\n'
        svg += f'<text x="{x_base + bar_width/2}" y="{svg_height - margin_bottom - h_b - 5}" text-anchor="middle" font-size="11" fill="#E24A33">{int(baseline_vals[i])}</text>\n'
        
        # Experimental Bar
        h_o = (optimized_vals[i] / max_val) * chart_height
        svg += f'<rect x="{x_base + bar_width + 5}" y="{svg_height - margin_bottom - h_o}" width="{bar_width}" height="{h_o}" fill="#348ABD"/>\n'
        svg += f'<text x="{x_base + bar_width*1.5 + 5}" y="{svg_height - margin_bottom - h_o - 5}" text-anchor="middle" font-size="11" fill="#348ABD">{int(optimized_vals[i])}</text>\n'
        
        svg += f'<text x="{x_base + bar_width + 2.5}" y="{svg_height - margin_bottom + 20}" text-anchor="middle" font-size="12" fill="#666">{label}</text>\n'

    svg += f'<line x1="{margin_left}" y1="{svg_height - margin_bottom}" x2="{svg_width - margin_right}" y2="{svg_height - margin_bottom}" stroke="#ccc" stroke-width="2"/>\n'
    svg += '</svg>'
    with open(filename, 'w') as f: f.write(svg)

# CPU Load (seconds of sample time)
create_bar_chart(
    'docs/contention_scaling_plot.svg',
    'Read-Path CPU Load (50 parallel LISTs, 30s)',
    ['5k Pods', '10k Pods'],
    [1275, 1306],
    [1276, 1299],
    'Total CPU Samples (Seconds)'
)

# Mutex Contention (sample count)
create_bar_chart(
    'docs/mutex_contention_plot.svg',
    'API Server Mutex Contention (30s Window)',
    ['5k LIST', '10k LIST', 'Brutal WRITE'],
    [0.1, 0.1, 0.1], # Using 0.1 so bar is visible but effectively zero
    [0.1, 0.1, 0.1],
    'Significant Mutex Delays (Count)'
)
