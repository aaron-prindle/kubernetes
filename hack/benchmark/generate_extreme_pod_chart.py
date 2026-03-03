import math

def create_bar_chart(filename, title, master_val, exp_val, ylabel):
    svg_width = 600
    svg_height = 450
    margin_left = 100
    margin_bottom = 60
    margin_top = 80
    margin_right = 40
    
    chart_width = svg_width - margin_left - margin_right
    chart_height = svg_height - margin_top - margin_bottom
    
    max_val = max(master_val, exp_val) * 1.2
    
    def get_y(val):
        return svg_height - margin_bottom - (val / max_val) * chart_height

    svg = f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {svg_width} {svg_height}" style="background-color: white; font-family: sans-serif;">
    <!-- Title -->
    <text x="{svg_width/2}" y="40" text-anchor="middle" font-size="20" font-weight="bold" fill="#333">{title}</text>
    
    <!-- Y Axis Label -->
    <text x="-{svg_height/2}" y="30" transform="rotate(-90)" text-anchor="middle" font-size="16" fill="#666">{ylabel}</text>
    
    <!-- Axes -->
    <line x1="{margin_left}" y1="{svg_height - margin_bottom}" x2="{svg_width - margin_right}" y2="{svg_height - margin_bottom}" stroke="#ccc" stroke-width="2"/>
    <line x1="{margin_left}" y1="{margin_top}" x2="{margin_left}" y2="{svg_height - margin_bottom}" stroke="#ccc" stroke-width="2"/>
    
    <!-- Legend -->
    <rect x="{svg_width/2 - 120}" y="60" width="15" height="15" fill="#E24A33" rx="2" ry="2"/>
    <text x="{svg_width/2 - 100}" y="72" font-size="14" fill="#333">master ([]byte)</text>
    <rect x="{svg_width/2 + 20}" y="60" width="15" height="15" fill="#348ABD" rx="2" ry="2"/>
    <text x="{svg_width/2 + 40}" y="72" font-size="14" fill="#333">experimental (stringhandle)</text>
    '''
    
    # Y-axis grid lines
    y_ticks = 5
    for i in range(y_ticks + 1):
        val = max_val * (i / y_ticks)
        y = get_y(val)
        svg += f'<line x1="{margin_left}" y1="{y}" x2="{svg_width - margin_right}" y2="{y}" stroke="#eee" stroke-width="1"/>\n'
        svg += f'<text x="{margin_left - 10}" y="{y + 5}" text-anchor="end" font-size="12" fill="#666">{int(val)}</text>\n'

    # Bars
    bar_width = 100
    spacing = 150
    center = margin_left + chart_width / 2

    # Master Bar
    x1 = center - spacing/2 - bar_width/2
    y1 = get_y(master_val)
    h1 = (master_val / max_val) * chart_height
    svg += f'<rect x="{x1}" y="{y1}" width="{bar_width}" height="{h1}" fill="#E24A33" rx="4" ry="4"/>\n'
    svg += f'<text x="{x1 + bar_width/2}" y="{y1 - 10}" text-anchor="middle" font-size="14" font-weight="bold" fill="#333">{master_val:.2f} MB</text>\n'

    # Experimental Bar
    x2 = center + spacing/2 - bar_width/2
    y2 = get_y(exp_val)
    h2 = (exp_val / max_val) * chart_height
    svg += f'<rect x="{x2}" y="{y2}" width="{bar_width}" height="{h2}" fill="#348ABD" rx="4" ry="4"/>\n'
    svg += f'<text x="{x2 + bar_width/2}" y="{y2 - 10}" text-anchor="middle" font-size="14" font-weight="bold" fill="#333">{exp_val:.2f} MB</text>\n'

    # X Labels
    svg += f'<text x="{x1 + bar_width/2}" y="{svg_height - margin_bottom + 25}" text-anchor="middle" font-size="14" font-weight="bold" fill="#333">Baseline</text>\n'
    svg += f'<text x="{x2 + bar_width/2}" y="{svg_height - margin_bottom + 25}" text-anchor="middle" font-size="14" font-weight="bold" fill="#333">Optimized</text>\n'

    svg += '</svg>'
    
    with open(filename, 'w') as f:
        f.write(svg)

# Data points from the extremely complex 20k run
master_total = 2376.66
exp_total = 2042.38

create_bar_chart(
    'docs/extreme_pod_memory_plot.svg',
    'Total Memory Reduction (20,000 Extreme Pods)',
    master_total,
    exp_total,
    'Total API Server Heap (MB)'
)

print("Extreme Pod Bar chart SVG generated successfully.")
