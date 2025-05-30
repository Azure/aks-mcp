#!/usr/bin/env python3
"""
Script to download AKS API specs from GitHub.
This script downloads the AKS API specs from the Azure REST API specs repository
and saves them to the internal/azure/spec directory.

Usage:
    python3 scripts/download_aks_specs.py [--dry-run]

Options:
    --dry-run   Print what would be downloaded without actually downloading files
"""

import os
import sys
import requests
import json
from pathlib import Path

# Base URLs for the GitHub repositories
BASE_URL = "https://raw.githubusercontent.com/Azure/azure-rest-api-specs/main"
SPEC_PATH = "specification/containerservice/resource-manager/Microsoft.ContainerService/aks/stable/2025-03-01"

# Use relative path from the script's location to the target directory
SCRIPT_DIR = Path(os.path.dirname(os.path.realpath(__file__)))
PROJECT_ROOT = SCRIPT_DIR.parent  # Parent of the scripts directory
TARGET_DIR = PROJECT_ROOT / "azure_spec"

def main():
    """Main function to download the AKS API specs"""
    # Parse command line arguments
    dry_run = "--dry-run" in sys.argv
    
    print(f"Target directory: {TARGET_DIR}")
    if dry_run:
        print("Dry run mode: files will not be downloaded")
    
    # Create target directories if not in dry run mode
    if not dry_run:
        os.makedirs(TARGET_DIR, exist_ok=True)
        os.makedirs(TARGET_DIR / "examples", exist_ok=True)

    print("Downloading AKS API specs from GitHub...")

    # Download the main spec file
    print("Downloading managedClusters.json...")
    main_spec_url = f"{BASE_URL}/{SPEC_PATH}/managedClusters.json"
    download_file(main_spec_url, TARGET_DIR / "managedClusters.json")

    # Get the list of example files
    print("Getting list of example files...")
    examples_api_url = f"https://api.github.com/repos/Azure/azure-rest-api-specs/contents/{SPEC_PATH}/examples"
    
    try:
        response = requests.get(examples_api_url)
        response.raise_for_status()
        examples = response.json()
        
        # Download each example file
        print("Downloading example files...")
        file_count = 0
        for example in examples:
            if example["type"] == "file":
                filename = example["name"]
                download_url = example["download_url"]
                file_count += 1
                print(f"  - {filename}")
                download_file(download_url, TARGET_DIR / "examples" / filename)
        
        dry_run_message = " (dry run - no files were actually downloaded)" if "--dry-run" in sys.argv else ""
        print(f"Download completed{dry_run_message}. {file_count + 1} files would be downloaded to {TARGET_DIR}/")
        return 0
    except requests.exceptions.RequestException as e:
        print(f"Error: Failed to get list of example files: {e}", file=sys.stderr)
        return 1

def download_file(url, dest_path):
    """Download a file from the given URL to the destination path"""
    # Skip actual download if in dry run mode
    if "--dry-run" in sys.argv:
        return
        
    try:
        response = requests.get(url)
        response.raise_for_status()
        with open(dest_path, "wb") as f:
            f.write(response.content)
    except requests.exceptions.RequestException as e:
        print(f"Error: Failed to download {url}: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    print(f"Script arguments: {sys.argv}")
    print(f"Project root: {PROJECT_ROOT}")
    print(f"Target directory: {TARGET_DIR}")
    sys.exit(main())
