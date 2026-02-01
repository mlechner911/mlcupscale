#!/usr/bin/env python3
import onnx
import sys
import os

def repack(file_path):
    # Use absolute path to avoid CWD confusion
    abs_path = os.path.abspath(file_path)
    dir_name = os.path.dirname(abs_path)
    base_name = os.path.basename(abs_path)

    print(f"Repacking {base_name} in {dir_name}...")

    cwd = os.getcwd()
    try:
        # Change to the directory of the model to ensure relative paths in ONNX work
        os.chdir(dir_name)

        # Load
        model = onnx.load(base_name, load_external_data=True)

        # Save (embedding data)
        onnx.save(model, base_name)

        size = os.path.getsize(base_name)
        print(f"Success! Saved {base_name} (Size: {size/1024/1024:.2f} MB)")

        # Remove external data
        data_file = base_name + ".data"
        if os.path.exists(data_file):
            print(f"Removing old external data file: {data_file}")
            os.remove(data_file)

    except Exception as e:
        print(f"Error repacking {file_path}: {e}")
    finally:
        # Always restore CWD
        os.chdir(cwd)

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python3 repack_models.py <model1.onnx> [model2.onnx ...]")
        sys.exit(1)

    for f in sys.argv[1:]:
        repack(f)
