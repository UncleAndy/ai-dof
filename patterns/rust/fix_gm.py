#!/usr/bin/env python3
"""Fix Rust graph_mapper.rs resources construction."""
path = "/home/hermes/projects/AI/DOF/patterns/rust/graph_mapper.rs"
with open(path) as f:
    content = f.read()

old = """        SystemStateMatrix {
            global_time_to_collapse_mks: global_ttc,
            context_switch_cost: self.context_switch_cost,
            entities,
            psi: Some(reference),
            resources: means.iter().map(|(k, v)| (k.clone(), *v)).collect(),
        }"""

new = """        SystemStateMatrix {
            global_time_to_collapse_mks: global_ttc,
            context_switch_cost: self.context_switch_cost,
            entities,
            psi: Some(reference),
            resources,
        }"""

if old in content:
    content = content.replace(old, new)
    with open(path, 'w') as f:
        f.write(content)
    print("Fixed graph_mapper.rs")
else:
    print("Pattern not found")
