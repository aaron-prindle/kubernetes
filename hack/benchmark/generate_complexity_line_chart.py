import math

def create_line_chart(filename, title, config_fields_data, baseline_vals, optimized_vals, ylabel):
    svg_width = 800
    svg_height = 500
    margin_left = 100
    margin_bottom = 80
    margin_top = 80
    margin_right = 40
    
    chart_width = svg_width - margin_left - margin_right
    chart_height = svg_height - margin_top - margin_bottom
    
    max_fields = max(config_fields_data)
    min_fields = min(config_fields_data)
    max_val = max(max(baseline_vals), max(optimized_vals)) * 1.1
    min_val = 0
    
    def get_x(fields):
        if max_fields == min_fields: return margin_left + chart_width / 2
        return margin_left + ((fields - min_fields) / (max_fields - min_fields)) * chart_width
        
    def get_y(val):
        return svg_height - margin_bottom - ((val - min_val) / (max_val - min_val)) * chart_height

    svg = f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {svg_width} {svg_height}" style="background-color: white; font-family: sans-serif;">
    <!-- Title -->
    <text x="{svg_width/2}" y="40" text-anchor="middle" font-size="22" font-weight="bold" fill="#333">{title}</text>
    
    <!-- Y Axis Label -->
    <text x="-{svg_height/2}" y="30" transform="rotate(-90)" text-anchor="middle" font-size="16" fill="#666">{ylabel}</text>
    
    <!-- X Axis Label -->
    <text x="{svg_width/2}" y="{svg_height - 20}" text-anchor="middle" font-size="16" fill="#666">Additional Config Fields per Pod</text>
    
    <!-- Legend -->
    <line x1="{svg_width/2 - 140}" y1="65" x2="{svg_width/2 - 110}" y2="65" stroke="#E24A33" stroke-width="3"/>
    <circle cx="{svg_width/2 - 125}" cy="65" r="5" fill="#E24A33"/>
    <text x="{svg_width/2 - 100}" y="70" font-size="14" fill="#333">master ([]byte)</text>
    
    <line x1="{svg_width/2 + 30}" y1="65" x2="{svg_width/2 + 60}" y2="65" stroke="#348ABD" stroke-width="3"/>
    <circle cx="{svg_width/2 + 45}" cy="65" r="5" fill="#348ABD"/>
    <text x="{svg_width/2 + 70}" y="70" font-size="14" fill="#333">experimental (stringhandle)</text>
    
    <!-- Grid and Axes -->
    '''
    
    y_ticks = 5
    for i in range(y_ticks + 1):
        val = min_val + (max_val - min_val) * (i / y_ticks)
        y = get_y(val)
        svg += f'<line x1="{margin_left}" y1="{y}" x2="{svg_width - margin_right}" y2="{y}" stroke="#eee" stroke-width="1"/>\n'
        svg += f'<text x="{margin_left - 10}" y="{y + 5}" text-anchor="end" font-size="12" fill="#666">{int(val)}</text>\n'

    for fields in config_fields_data:
        x = get_x(fields)
        svg += f'<line x1="{x}" y1="{svg_height - margin_bottom}" x2="{x}" y2="{svg_height - margin_bottom + 5}" stroke="#ccc" stroke-width="2"/>\n'
        svg += f'<text x="{x}" y="{svg_height - margin_bottom + 20}" text-anchor="middle" font-size="12" fill="#666">+{fields}</text>\n'

    svg += f'''
    <line x1="{margin_left}" y1="{svg_height - margin_bottom}" x2="{svg_width - margin_right}" y2="{svg_height - margin_bottom}" stroke="#ccc" stroke-width="2"/>
    <line x1="{margin_left}" y1="{margin_top}" x2="{margin_left}" y2="{svg_height - margin_bottom}" stroke="#ccc" stroke-width="2"/>
    '''

    points_baseline = [(get_x(p), get_y(v)) for p, v in zip(config_fields_data, baseline_vals)]
    points_optimized = [(get_x(p), get_y(v)) for p, v in zip(config_fields_data, optimized_vals)]
    
    path_d = f"M {points_baseline[0][0]} {points_baseline[0][1]} "
    for p in points_baseline[1:]:
        path_d += f"L {p[0]} {p[1]} "
    for p in reversed(points_optimized):
        path_d += f"L {p[0]} {p[1]} "
    path_d += "Z"
    
    svg += f'<path d="{path_d}" fill="#348ABD" opacity="0.15" />\n'

    path_b = f"M {points_baseline[0][0]} {points_baseline[0][1]} "
    for p in points_baseline[1:]: path_b += f"L {p[0]} {p[1]} "
    svg += f'<path d="{path_b}" fill="none" stroke="#E24A33" stroke-width="3" />\n'
    
    path_o = f"M {points_optimized[0][0]} {points_optimized[0][1]} "
    for p in points_optimized[1:]: path_o += f"L {p[0]} {p[1]} "
    svg += f'<path d="{path_o}" fill="none" stroke="#348ABD" stroke-width="3" />\n'

    for i, (fields, val) in enumerate(zip(config_fields_data, baseline_vals)):
        x, y = get_x(fields), get_y(val)
        svg += f'<circle cx="{x}" cy="{y}" r="6" fill="#E24A33" stroke="white" stroke-width="2"/>\n'
        if val > 0:
            svg += f'<text x="{x}" y="{y - 12}" text-anchor="middle" font-size="12" font-weight="bold" fill="#E24A33">{val:.0f}</text>\n'

    for i, (fields, val) in enumerate(zip(config_fields_data, optimized_vals)):
        x, y = get_x(fields), get_y(val)
        svg += f'<circle cx="{x}" cy="{y}" r="6" fill="#348ABD" stroke="white" stroke-width="2"/>\n'
        if val > 0:
            y_offset = 20 if fields < max_fields else 25
            if i > 0 and (baseline_vals[i] - val) < 200:
                y_offset = 20
            svg += f'<text x="{x}" y="{y + y_offset}" text-anchor="middle" font-size="12" font-weight="bold" fill="#348ABD">{val:.0f}</text>\n'

    svg += '</svg>'
    
    with open(filename, 'w') as f:
        f.write(svg)

# Data points across all 20k runs
config_fields = [0, 75, 150, 300, 600, 1200]
# Master Total Heap
baseline = [733.25, 1636.51, 2376.66, 3036.72, 4834.35, 7196.65]
# Optimized Total Heap
optimized = [719.85, 1454.69, 2042.38, 2475.58, 3947.89, 5823.93]

create_line_chart(
    'docs/complexity_scaling_plot.svg',
    'Total Heap Scaling by Pod Complexity (20,000 Pods)',
    config_fields,
    baseline,
    optimized,
    'Total API Server Heap (MB)'
)

print("Complexity line chart SVG generated successfully.")
