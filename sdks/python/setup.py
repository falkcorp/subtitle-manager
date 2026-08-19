# file: sdks/python/setup.py
# version: 1.0.1
# guid: 550e8400-e29b-41d4-a716-446655440008

"""
Setup configuration for the Subtitle Manager Python SDK.
"""

from setuptools import setup, find_packages
import os


# Read README file
def read_readme():
    """Read README file for long description."""
    readme_path = os.path.join(os.path.dirname(__file__), "README.md")
    if os.path.exists(readme_path):
        with open(readme_path, "r", encoding="utf-8") as f:
            return f.read()
    return ""


# Read requirements
def read_requirements():
    """Read requirements from requirements.txt."""
    req_path = os.path.join(os.path.dirname(__file__), "requirements.txt")
    if os.path.exists(req_path):
        with open(req_path, "r", encoding="utf-8") as f:
            return [
                line.strip() for line in f if line.strip() and not line.startswith("#")
            ]
    return ["requests>=2.25.0"]


setup(
    name="subtitle-manager-sdk",
    version="1.0.0",
    author="Subtitle Manager Team",
    author_email="support@subtitlemanager.com",
    description="Python SDK for Subtitle Manager API",
    long_description=read_readme(),
    long_description_content_type="text/markdown",
    url="https://github.com/jdfalk/subtitle-manager",
    project_urls={
        "Bug Reports": "https://github.com/jdfalk/subtitle-manager/issues",
        "Source": "https://github.com/jdfalk/subtitle-manager",
        "Documentation": "https://github.com/jdfalk/subtitle-manager/tree/main/docs",
    },
    packages=find_packages(),
    classifiers=[
        "Development Status :: 4 - Beta",
        "Intended Audience :: Developers",
        "License :: Other/Proprietary License",
        "Operating System :: OS Independent",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
        "Programming Language :: Python :: 3.12",
        "Topic :: Software Development :: Libraries :: Python Modules",
        "Topic :: Multimedia :: Video",
        "Topic :: Text Processing",
    ],
    python_requires=">=3.8",
    install_requires=read_requirements(),
    extras_require={
        "dev": [
            "pytest>=6.0",
            "pytest-cov>=2.0",
            "pytest-mock>=3.0",
            "black>=21.0",
            "flake8>=3.8",
            "mypy>=0.900",
            "responses>=0.18.0",
        ],
        "test": [
            "pytest>=6.0",
            "pytest-cov>=2.0",
            "pytest-mock>=3.0",
            "responses>=0.18.0",
        ],
    },
    keywords=[
        "subtitle",
        "translation",
        "media",
        "api",
        "sdk",
        "subtitles",
        "srt",
        "video",
    ],
    include_package_data=True,
    zip_safe=False,
)
