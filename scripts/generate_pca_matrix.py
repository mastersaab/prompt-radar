#!/usr/bin/env python3
"""
generate_pca_matrix.py
Generates a deterministic 768x2 static projection matrix W for PromptRadar.
This maps high-dimensional 768-D embeddings to fixed, non-drifting 2D screen coordinates [-1.0, 1.0].
"""

import json
import math
import random
import os

SEED = 42
DIMS = 768
OUTPUT_COLS = 2

def generate_projection_matrix(dims=DIMS, cols=OUTPUT_COLS, seed=SEED):
    rng = random.Random(seed)
    matrix = []
    
    # Generate Gaussian random matrix (Johnson-Lindenstrauss projection / PCA initial base)
    col_sums_sq = [0.0] * cols
    raw_matrix = []
    for i in range(dims):
        row = []
        for j in range(cols):
            # Box-Muller transform for normal distribution
            u1 = max(1e-9, rng.random())
            u2 = rng.random()
            val = math.sqrt(-2.0 * math.log(u1)) * math.cos(2.0 * math.pi * u2)
            row.append(val)
            col_sums_sq[j] += val * val
        raw_matrix.append(row)
    
    # Gram-Schmidt orthogonalization to ensure columns 0 and 1 are orthogonal
    # Normalize col 0
    norm0 = math.sqrt(col_sums_sq[0])
    for i in range(dims):
        raw_matrix[i][0] /= norm0
        
    # Subtract projection of col 1 onto col 0
    dot01 = sum(raw_matrix[i][0] * raw_matrix[i][1] for i in range(dims))
    for i in range(dims):
        raw_matrix[i][1] -= dot01 * raw_matrix[i][0]
        
    # Normalize col 1
    norm1 = math.sqrt(sum(raw_matrix[i][1] ** 2 for i in range(dims)))
    for i in range(dims):
        raw_matrix[i][1] /= norm1
        
    # Scale for 2D screen visual bounding [-0.85, 0.85] on unit vectors
    scale = 2.4 / math.sqrt(dims)
    for i in range(dims):
        matrix.append([
            round(raw_matrix[i][0] * scale, 6),
            round(raw_matrix[i][1] * scale, 6)
        ])
        
    return matrix

if __name__ == "__main__":
    matrix = generate_projection_matrix()
    out_dir = os.path.dirname(os.path.abspath(__file__))
    out_path = os.path.join(out_dir, "projection_matrix.json")
    with open(out_path, "w") as f:
        json.dump({
            "dims": DIMS,
            "out_dimensions": OUTPUT_COLS,
            "matrix": matrix
        }, f, indent=2)
    print(f"Generated static {DIMS}x{OUTPUT_COLS} projection matrix -> {out_path}")
