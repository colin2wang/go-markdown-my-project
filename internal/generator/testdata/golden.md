# Project Documentation for Markdown My Project

## Project File Tree

```
Markdown My Project
├── cargo.toml
├── projects/
│   ├── epub_reader.yml
│   ├── project1.yml
│   └── project2.yml
└── src/
    ├── config.rs
    ├── file_processor.rs
    ├── language.rs
    ├── logger.rs
    ├── main.rs
    ├── markdown_generator.rs
    └── tree_generator.rs
```

## Project Files

### File: `cargo.toml`

```Text
[package]
name = "markdown_my_project"
version = "0.1.0"
edition = "2021"

[dependencies]
serde = { version = "1.0", features = ["derive"] }
serde_yaml = "0.9"
log = "0.4.26"
log4rs = "1.3.0"
anyhow = "1.0"
clap = { version = "4.0", features = ["derive"] }
walkdir = "2.3"
glob = "0.3"
indicatif = "0.17"
rayon = "1.7"

[dev-dependencies]
tempfile = "3.10"
```

### File: `projects\epub_reader.yml`

```YAML
# Project Configuration for Project Documentation

# Name of the project
project_name: "EPUB Reader"

# Path to the project root directory
project_path: "C:/Workspaces/JetBrains/AndroidStudio/epub_reader"

# Output file path for the generated documentation
output_file: "epub_reader.md"

# Markdown output language: "zh_cn" for Chinese, "en_us" for English
markdown_lang: zh_cn

# List of specific files to include in the documentation
files:

# List of directories to include in the documentation (files within these directories will be processed recursively)
directories:
  - lib

# List of directories to exclude from the documentation
exclude_directories:
  - "**/444"
```

### File: `projects\project1.yml`

```YAML
# Project Configuration for Project Documentation

# Name of the project
project_name: "Markdown My Project"

# Path to the project root directory
project_path: "F:/Workspaces/JetBrains/RustRover/markdown_my_project"

# Output file path for the generated documentation
output_file: "markdown_my_project.md"

# Markdown output language: "zh_cn" for Chinese, "en_us" for English
markdown_lang: zh_cn

# List of specific files to include in the documentation
files:
  - cargo.toml
  - log4rs.yml
  - languages.yml

# List of directories to include in the documentation (files within these directories will be processed recursively)
directories:
  - src
  - test
  - projects

# List of directories to exclude from the documentation
exclude_directories:
  - "**/444"
```

### File: `projects\project2.yml`

```YAML
# Project Configuration for Project Documentation

# Name of the project
project_name: "PDF to Excel"

# Path to the project root directory
project_path: "F:/Workspaces/JetBrains/RustRover/pdf_to_excel"

# Output file path for the generated documentation
output_file: "pdf_to_excel.md"

# Markdown output language: "zh_cn" for Chinese, "en_us" for English
markdown_lang: en_us

# List of specific files to include in the documentation
files:
  - cargo.toml
  - config.yml
  - log4rs.yml

# List of directories to include in the documentation (files within these directories will be processed recursively)
directories:
  - src
  - test
  - projects

# List of directories to exclude from the documentation
exclude_directories:
```

### File: `src\config.rs`

```Rust
use serde::Deserialize;
use std::fs;
use std::path::PathBuf;
use anyhow::{Context, Result};

/// Configuration structure for a project.
///
/// This struct represents the configuration for a project, including its name,
/// root directory path, output file path, specific files to include, directories
/// to include recursively, and directories to exclude.
#[derive(Debug, Deserialize)]
pub struct Config {
    /// Name of the project.
    pub project_name: String,
    /// Path to the project root directory.
    pub project_path: PathBuf,
    /// Path to the output file where the documentation will be saved.
    pub output_file: PathBuf,
    /// List of specific files to include in the documentation.
    pub files: Vec<PathBuf>,
    /// List of directories to include in the documentation (files within these directories will be processed recursively).
    pub directories: Vec<PathBuf>,
    /// List of directories to exclude from the documentation.
    #[serde(default)]
    pub exclude_directories: Vec<String>,
    /// List of glob patterns to exclude files (e.g., "*.log", "target/**").
    #[serde(default)]
    pub exclude_patterns: Vec<String>,
    /// Maximum file size in bytes. Files larger than this will be skipped.
    /// If not specified, no limit is applied.
    #[serde(default)]
    pub max_file_size: Option<u64>,

    /// Markdown output language. "zh_cn" for Chinese, "en_us" for English (default).
    #[serde(default = "default_markdown_lang")]
    pub markdown_lang: String,
}

fn default_markdown_lang() -> String {
    "en_us".to_string()
}

impl Config {
    /// Loads a project configuration from a YAML file.
    ///
    /// # Arguments
    ///
    /// * `config_path` - Path to the YAML configuration file.
    ///
    /// # Returns
    ///
    /// * `Result<Self>` - The loaded configuration or an error.
    pub fn load(config_path: &PathBuf) -> Result<Self> {
        let config_content = fs::read_to_string(config_path)
            .context(format!("Failed to read configuration file: {}", config_path.display()))?;
        
        let config: Config = serde_yaml::from_str(&config_content)
            .context(format!("Failed to parse configuration file: {}", config_path.display()))?;
        
        // Validate configuration
        config.validate()
            .context(format!("Invalid configuration in: {}", config_path.display()))?;
        
        Ok(config)
    }

    /// Validates the configuration values.
    fn validate(&self) -> Result<Self> {
        // Validate project name
        if self.project_name.trim().is_empty() {
            anyhow::bail!("Project name cannot be empty");
        }

        // Validate project path
        if !self.project_path.exists() {
            anyhow::bail!("Project path does not exist: {}", self.project_path.display());
        }
        if !self.project_path.is_dir() {
            anyhow::bail!("Project path is not a directory: {}", self.project_path.display());
        }

        // Validate output file extension
        if let Some(ext) = self.output_file.extension() {
            let ext_str = ext.to_string_lossy().to_lowercase();
            if !["md", "markdown", "html", "txt"].contains(&ext_str.as_str()) {
                log::warn!("Output file extension '{}' might not be supported", ext_str);
            }
        }

        // Validate max file size
        if let Some(max_size) = self.max_file_size {
            if max_size == 0 {
                anyhow::bail!("Max file size cannot be zero");
            }
        }

        // Validate exclude patterns
        for pattern in &self.exclude_patterns {
            if pattern.trim().is_empty() {
                anyhow::bail!("Exclude pattern cannot be empty");
            }
        }

        Ok(self.clone())
    }
}

impl Clone for Config {
    fn clone(&self) -> Self {
        Config {
            project_name: self.project_name.clone(),
            project_path: self.project_path.clone(),
            output_file: self.output_file.clone(),
            files: self.files.clone(),
            directories: self.directories.clone(),
            exclude_directories: self.exclude_directories.clone(),
            exclude_patterns: self.exclude_patterns.clone(),
            max_file_size: self.max_file_size,
            markdown_lang: self.markdown_lang.clone(),
        }
    }
}

```

### File: `src\file_processor.rs`

```Rust
use std::fs;
use std::io::Read;
use std::path::{Path, PathBuf};
use std::sync::{Arc, Mutex};
use anyhow::{Context, Result};
use rayon::prelude::*;
use walkdir::WalkDir;

/// Reads the content of a file.
///
/// # Arguments
///
/// * `file_path` - Path to the file to read.
///
/// # Returns
///
/// * `Result<String>` - The content of the file or an error.
pub fn read_file_content(file_path: &Path) -> Result<String> {
    let mut file = fs::File::open(file_path)
        .context(format!("Failed to open file: {}", file_path.display()))?;
    let mut content = String::new();
    file.read_to_string(&mut content)
        .context(format!("Failed to read file: {}", file_path.display()))?;
    Ok(content)
}

/// Processes files and directories specified in the configuration.
///
/// This function processes individual files and directories recursively, excluding
/// directories and files specified in the configuration.
///
/// # Arguments
///
/// * `project_path` - Path to the project root directory.
/// * `files` - List of specific files to process.
/// * `directories` - List of directories to process recursively.
/// * `exclude_directories` - List of directories to exclude from processing.
/// * `exclude_patterns` - List of glob patterns to exclude files.
/// * `max_file_size` - Maximum file size in bytes (optional).
///
/// # Returns
///
/// * `Result<Vec<(PathBuf, String)>>` - A list of file paths and their contents.
pub fn process_files(
    project_path: &PathBuf,
    files: &[PathBuf],
    directories: &[PathBuf],
    exclude_directories: &[String],
    exclude_patterns: &[String],
    max_file_size: Option<u64>,
) -> Result<Vec<(PathBuf, String)>> {
    let mut file_contents = Vec::new();

    // Process individual files
    for file in files {
        let full_path = project_path.join(file);
        if full_path.exists() && full_path.is_file() {
            if should_include_file(&full_path, exclude_patterns, max_file_size, project_path)? {
                let content = read_file_content(&full_path)?;
                file_contents.push((full_path, content));
            }
        }
    }

    // Process files within directories recursively
    for dir in directories {
        let full_dir = project_path.join(dir);
        if full_dir.exists() && full_dir.is_dir() {
            process_directory_parallel(&full_dir, &mut file_contents, exclude_directories, exclude_patterns, max_file_size, project_path)?;
        }
    }

    Ok(file_contents)
}

/// Processes files within a directory in parallel using rayon.
///
/// # Arguments
///
/// * `dir` - Path to the directory to process.
/// * `file_contents` - Vector to store file paths and their contents.
/// * `exclude_directories` - List of directories to exclude from processing.
/// * `exclude_patterns` - List of glob patterns to exclude files.
/// * `max_file_size` - Maximum file size in bytes (optional).
/// * `project_root` - Path to the project root directory.
///
/// # Returns
///
/// * `Result<()>` - Success or error.
fn process_directory_parallel(
    dir: &Path,
    file_contents: &mut Vec<(PathBuf, String)>,
    exclude_directories: &[String],
    exclude_patterns: &[String],
    max_file_size: Option<u64>,
    project_root: &Path,
) -> Result<()> {
    // Collect all file paths first
    let file_paths: Vec<PathBuf> = WalkDir::new(dir)
        .into_iter()
        .filter_entry(|e| !should_exclude_directory(e.path(), exclude_directories))
        .filter_map(|e| e.ok())
        .filter(|e| e.file_type().is_file())
        .map(|e| e.path().to_path_buf())
        .collect();

    // Process files in parallel
    let file_contents_mutex = Arc::new(Mutex::new(Vec::new()));
    
    file_paths.par_iter()
        .filter_map(|path| {
            if should_include_file(path, exclude_patterns, max_file_size, project_root).unwrap_or(false) {
                match read_file_content(path) {
                    Ok(content) => Some((path.clone(), content)),
                    Err(e) => {
                        log::warn!("Failed to read file {}: {}", path.display(), e);
                        None
                    }
                }
            } else {
                None
            }
        })
        .for_each(|(path, content)| {
            file_contents_mutex.lock().unwrap().push((path, content));
        });

    let mut collected = Arc::try_unwrap(file_contents_mutex)
        .unwrap()
        .into_inner()
        .unwrap();
    
    file_contents.append(&mut collected);
    Ok(())
}

/// Determines whether a file should be included based on exclude patterns and size limit.
///
/// # Arguments
///
/// * `file_path` - Path to the file to check.
/// * `exclude_patterns` - List of glob patterns to exclude files.
/// * `max_file_size` - Maximum file size in bytes (optional).
/// * `project_root` - Path to the project root directory.
///
/// # Returns
///
/// * `Result<bool>` - `true` if the file should be included, `false` otherwise.
fn should_include_file(
    file_path: &Path,
    exclude_patterns: &[String],
    max_file_size: Option<u64>,
    project_root: &Path,
) -> Result<bool> {
    // Check file size limit
    if let Some(max_size) = max_file_size {
        let metadata = fs::metadata(file_path)
            .context(format!("Failed to get metadata for: {}", file_path.display()))?;
        if metadata.len() > max_size {
            log::debug!("Skipping large file: {} ({} bytes > {} bytes)", 
                file_path.display(), metadata.len(), max_size);
            return Ok(false);
        }
    }

    // Check exclude patterns
    let relative_path = file_path.strip_prefix(project_root).unwrap_or(file_path);
    let relative_path_str = relative_path.to_string_lossy();
    
    for pattern in exclude_patterns {
        if pattern.contains('*') || pattern.contains('?') {
            // Use glob pattern matching
            if let Ok(compiler) = glob::Pattern::new(pattern) {
                if compiler.matches(&relative_path_str) {
                    log::debug!("Skipping file due to pattern '{}': {}", pattern, file_path.display());
                    return Ok(false);
                }
            }
        } else {
            // Exact match or directory match
            if *relative_path_str == *pattern || relative_path_str.starts_with(pattern.as_str()) {
                log::debug!("Skipping file due to pattern '{}': {}", pattern, file_path.display());
                return Ok(false);
            }
        }
    }

    Ok(true)
}

/// Determines whether a directory should be excluded based on the `exclude_directories` list.
///
/// # Arguments
///
/// * `dir` - Path to the directory to check.
/// * `exclude_directories` - List of directories to exclude.
///
/// # Returns
///
/// * `bool` - `true` if the directory should be excluded, `false` otherwise.
fn should_exclude_directory(dir: &Path, exclude_directories: &[String]) -> bool {
    for pattern in exclude_directories {
        if pattern == "**" {
            return true;
        } else if pattern.starts_with("**/") {
            let dir_name_to_exclude = &pattern[3..];
            let current_dir_name = dir.file_name()
                .and_then(|os_str| os_str.to_str())
                .unwrap_or("");
            if current_dir_name == dir_name_to_exclude {
                return true;
            }
        } else if pattern.contains('/') || pattern.contains('\\') {
            // Pattern contains path separator, match against relative path
            let rel_path = dir.strip_prefix(Path::new(".")).unwrap_or(dir);
            if rel_path == Path::new(pattern) {
                return true;
            }
        } else {
            // Pattern is just a directory name, match against any component
            let current_dir_name = dir.file_name()
                .and_then(|os_str| os_str.to_str())
                .unwrap_or("");
            if current_dir_name == pattern {
                return true;
            }
            
            // Also check if any parent component matches
            for component in dir.components() {
                if let std::path::Component::Normal(name) = component {
                    if *name.to_string_lossy() == *pattern {
                        return true;
                    }
                }
            }
        }
    }
    false
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs::File;
    use std::io::Write;
    use tempfile::TempDir;

    #[test]
    fn test_read_file_content() {
        let temp_dir = TempDir::new().unwrap();
        let file_path = temp_dir.path().join("test.txt");
        let mut file = File::create(&file_path).unwrap();
        writeln!(file, "Hello, World!").unwrap();

        let content = read_file_content(&file_path).unwrap();
        assert_eq!(content, "Hello, World!\n");
    }

    #[test]
    fn test_should_exclude_directory() {
        let dir = Path::new("target/debug");
        assert!(should_exclude_directory(dir, &["target".to_string()]));
        assert!(should_exclude_directory(dir, &["**/debug".to_string()]));
        assert!(!should_exclude_directory(dir, &["src".to_string()]));
    }

    #[test]
    fn test_should_include_file() {
        let temp_dir = TempDir::new().unwrap();
        let file_path = temp_dir.path().join("test.txt");
        let mut file = File::create(&file_path).unwrap();
        writeln!(file, "Hello").unwrap();

        // No exclusions
        assert!(should_include_file(&file_path, &[], None, temp_dir.path()).unwrap());

        // With exclude pattern
        assert!(!should_include_file(&file_path, &["*.txt".to_string()], None, temp_dir.path()).unwrap());

        // With size limit (small file)
        assert!(should_include_file(&file_path, &[], Some(1024), temp_dir.path()).unwrap());

        // With size limit (file too large)
        assert!(!should_include_file(&file_path, &[], Some(1), temp_dir.path()).unwrap());
    }
}

```

### File: `src\language.rs`

```Rust
use serde::Deserialize;
use std::collections::HashMap;
use std::fs;
use anyhow::{Context, Result};

/// Loads language definitions from a YAML file.
///
/// # Arguments
///
/// * `languages_path` - Path to the YAML file containing language definitions.
///
/// # Returns
///
/// * `Result<HashMap<String, String>>` - A map of file extensions to language names.
pub fn load_languages(languages_path: &std::path::Path) -> Result<HashMap<String, String>> {
    let content = fs::read_to_string(languages_path)
        .context(format!("Failed to read languages file: {}", languages_path.display()))?;
    
    let config: LanguageConfig = serde_yaml::from_str(&content)
        .context(format!("Failed to parse languages file: {}", languages_path.display()))?;
    
    Ok(config.languages)
}

#[derive(Deserialize)]
struct LanguageConfig {
    languages: HashMap<String, String>,
}
```

### File: `src\logger.rs`

```Rust
use std::fs;
use anyhow::{Context, Result};

/// Initializes the logger configured by log4rs.
///
/// The logging configuration is loaded from `log4rs.yml`.
///
/// # Returns
///
/// * `Result<()>` - Success or error.
pub fn init_logger() -> Result<()> {
    // Ensure the logs directory exists
    fs::create_dir_all("logs")
        .context("Failed to create logs directory")?;

    // Load the log4rs configuration from the YAML file
    let config = log4rs::config::load_config_file("log4rs.yml", Default::default())
        .context("Failed to load log4rs configuration")?;

    // Initialize the logger with the loaded configuration
    log4rs::init_config(config)
        .context("Failed to initialize log4rs")?;

    log::info!("Logger initialized with log4rs configuration.");
    Ok(())
}
```

### File: `src\main.rs`

```Rust
mod config;
mod file_processor;
mod markdown_generator;
mod logger;
mod language;
mod tree_generator;

use std::fs;
use std::path::{Path, PathBuf};
use anyhow::{Context, Result};
use clap::Parser;
use indicatif::{ProgressBar, ProgressStyle};

#[derive(Parser, Debug)]
#[command(author, version, about, long_about = None)]
struct Args {
    /// Path to the projects directory
    #[arg(short, long, default_value = "projects")]
    projects_dir: PathBuf,

    /// Path to the languages definition file
    #[arg(short, long, default_value = "languages.yml")]
    languages_file: PathBuf,

    /// Output directory for generated documentation
    #[arg(short, long, default_value = "output")]
    output_dir: PathBuf,

    /// Enable verbose logging
    #[arg(short, long)]
    verbose: bool,
}

fn main() -> Result<()> {
    let args = Args::parse();

    // Initialize the logger with the configuration from log4rs.yml
    logger::init_logger().context("Failed to initialize logger")?;

    log::info!("Starting to generate project documentation...");

    // Load common language definitions from the YAML file
    let languages = language::load_languages(&args.languages_file)
        .context("Failed to load language definitions")?;
    log::info!("Loaded language definitions from {}", args.languages_file.display());

    // Ensure output directory exists
    fs::create_dir_all(&args.output_dir)
        .context("Failed to create output directory")?;

    // Get list of project configuration files
    let config_files = get_config_files(&args.projects_dir)?;
    let total_projects = config_files.len();

    if total_projects == 0 {
        log::warn!("No project configuration files found in {}", args.projects_dir.display());
        return Ok(());
    }

    // Create progress bar
    let pb = ProgressBar::new(total_projects as u64);
    pb.set_style(ProgressStyle::default_bar()
        .template("{spinner:.green} [{elapsed_precise}] [{bar:40.cyan/blue}] {pos}/{len} ({eta})")
        .unwrap()
        .progress_chars("#>-"));

    // Process each project configuration
    for config_path in config_files {
        let config_name = config_path.file_stem()
            .unwrap_or_default()
            .to_string_lossy();
        
        pb.set_message(format!("Processing: {}", config_name));

        // Load the project configuration from the YAML file
        let config = config::Config::load(&config_path)
            .context(format!("Failed to load configuration: {}", config_path.display()))?;
        log::info!("Loaded project configuration: {}", config.project_name);

        // Get the project root directory
        let project_root = Path::new(&config.project_path);

        // Process files and directories specified in the configuration
        let files = file_processor::process_files(
            &config.project_path,
            &config.files,
            &config.directories,
            &config.exclude_directories,
            &config.exclude_patterns,
            config.max_file_size,
        ).context(format!("Failed to process files for project: {}", config.project_name))?;
        log::info!("Processed {} files for project: {}", files.len(), config.project_name);

        // Generate Markdown content for the project documentation
        let markdown_content = markdown_generator::generate_markdown(
            &config.project_name,
            files,
            &languages,
            project_root,
            &config.markdown_lang,
        ).context(format!("Failed to generate markdown for project: {}", config.project_name))?;

        // Write the generated Markdown content to the output file with UTF-8 encoding
        let output_path = args.output_dir.join(&config.output_file);
        use std::io::Write;
        let mut file = fs::File::create(&output_path)
            .context(format!("Failed to create output file: {}", output_path.display()))?;
        file.write_all(markdown_content.as_bytes())
            .context(format!("Failed to write output file: {}", output_path.display()))?;
        log::info!("Generated documentation for project: {} -> {}", config.project_name, output_path.display());

        pb.inc(1);
    }

    pb.finish_with_message("Done");
    log::info!("Project documentation generation complete.");
    Ok(())
}

/// Get all YAML configuration files from the projects directory
fn get_config_files(projects_dir: &Path) -> Result<Vec<PathBuf>> {
    let mut config_files = Vec::new();
    
    if !projects_dir.exists() {
        return Ok(config_files);
    }

    for entry in fs::read_dir(projects_dir)
        .context(format!("Failed to read projects directory: {}", projects_dir.display()))? {
        let entry = entry.context("Failed to read directory entry")?;
        let config_path = entry.path();
        
        if config_path.is_file() && config_path.extension().unwrap_or_default() == "yml" {
            config_files.push(config_path);
        }
    }

    // Sort for consistent ordering
    config_files.sort();
    Ok(config_files)
}

```

### File: `src\markdown_generator.rs`

```Rust
use std::path::{Path, PathBuf};
use std::collections::HashMap;
use anyhow::Result;
use crate::tree_generator;

/// Returns the localized heading for a given key.
fn localized_text(key: &str, lang: &str) -> String {
    match (key, lang) {
        ("project_documentation", "zh_cn") => "项目文档".to_string(),
        ("project_documentation", _) => "Project Documentation".to_string(),
        ("project_file_tree", "zh_cn") => "项目文件树".to_string(),
        ("project_file_tree", _) => "Project File Tree".to_string(),
        ("project_files", "zh_cn") => "项目文件".to_string(),
        ("project_files", _) => "Project Files".to_string(),
        ("file_label", "zh_cn") => "文件".to_string(),
        ("file_label", _) => "File".to_string(),
        _ => key.to_string(),
    }
}

/// Generates Markdown documentation for a project based on its files and directories.
///
/// # Arguments
///
/// * `project_name` - Name of the project.
/// * `files` - List of files with their paths and contents.
/// * `languages` - Mapping of file extensions to language names.
/// * `project_root` - Path to the project root directory.
/// * `lang` - Output language: "zh_cn" or "en_us" (default).
///
/// # Returns
///
/// * `Result<String>` - The generated Markdown content or an error.
pub fn generate_markdown(
    project_name: &str,
    files: Vec<(PathBuf, String)>,
    languages: &HashMap<String, String>,
    project_root: &Path,
    lang: &str,
) -> Result<String> {
    // Sort files for consistent output
    let mut sorted_files = files;
    sorted_files.sort_by(|a, b| a.0.cmp(&b.0));

    let title = localized_text("project_documentation", lang);
    let mut markdown_content = format!("# {} for {}\n\n", title, project_name);

    // Add the project file tree to the Markdown (at the top)
    let tree_heading = localized_text("project_file_tree", lang);
    markdown_content.push_str(&format!("## {}\n\n", tree_heading));
    markdown_content.push_str("```\n");
    markdown_content.push_str(&format!("{}\n", project_name));
    markdown_content.push_str(&tree_generator::generate_tree(project_name, &sorted_files, project_root)?);
    markdown_content.push_str("```\n\n");

    // Add file contents to the Markdown
    let files_heading = localized_text("project_files", lang);
    let file_label = localized_text("file_label", lang);
    markdown_content.push_str(&format!("## {}\n\n", files_heading));
    for (file_path, content) in &sorted_files {
        // Get the relative path of the file with respect to the project root
        let relative_path = file_path.strip_prefix(project_root).unwrap_or(file_path);
        let display_path = relative_path.display();

        // Determine the file extension and corresponding language
        let extension = file_path
            .extension()
            .unwrap_or_default()
            .to_string_lossy()
            .to_lowercase();

        let language = languages.get(&extension).unwrap_or(&"Text".to_string()).clone();

        markdown_content.push_str(&format!(
            "### {}: `{}`\n\n```{}\n{}\n```\n\n",
            file_label,
            display_path,
            language,
            content
        ));
    }

    Ok(markdown_content)
}

```

### File: `src\tree_generator.rs`

```Rust
use std::path::{Path, PathBuf};
use std::collections::BTreeMap;
use anyhow::Result;

/// Represents a directory in the project tree.
#[derive(Debug)]
struct Directory {
    name: String,
    files: Vec<String>,
    subdirectories: BTreeMap<String, Directory>,
}

impl Directory {
    fn new(name: String) -> Self {
        Directory {
            name,
            files: Vec::new(),
            subdirectories: BTreeMap::new(),
        }
    }

    fn add_file(&mut self, file: String) {
        self.files.push(file);
    }
}

/// Builds a directory tree from a list of files.
///
/// # Arguments
///
/// * `files` - List of files with their paths and contents.
/// * `project_root` - Path to the project root directory.
///
/// # Returns
///
/// * `Directory` - The root directory of the tree.
fn build_directory_tree(files: &[(PathBuf, String)], project_root: &Path) -> Directory {
    let mut root = Directory::new("".to_string());

    for (file_path, _) in files {
        let relative_path = file_path.strip_prefix(project_root).unwrap_or(file_path);
        let components: Vec<String> = relative_path
            .components()
            .map(|c| c.as_os_str().to_string_lossy().into_owned())
            .collect();

        let mut current_dir = &mut root;
        for (i, component) in components.iter().enumerate() {
            if i < components.len() - 1 {
                let entry = current_dir.subdirectories.entry(component.clone());
                let sub_dir = entry.or_insert_with(|| Directory::new(component.clone()));
                current_dir = sub_dir;
            } else {
                current_dir.add_file(component.clone());
            }
        }
    }

    root
}

/// Converts a directory tree to a string representation.
///
/// # Arguments
///
/// * `directory` - The directory to convert.
/// * `indent` - Current indentation string.
/// * `is_last` - Whether this is the last item in its parent.
/// * `is_root` - Whether this is the root directory.
///
/// # Returns
///
/// * `String` - The string representation of the directory tree.
fn directory_tree_to_string(
    directory: &Directory,
    indent: &str,
    is_last: bool,
    is_root: bool,
) -> String {
    let mut tree = String::new();
    if !is_root {
        let connector = if is_last { "└── " } else { "├── " };
        let line = format!("{}{}{}/", indent, connector, directory.name);
        tree.push_str(&line);
        tree.push('\n');
    }

    // Calculate new indent based on whether current directory is last or root
    let new_indent = if is_root {
        indent.to_string()
    } else if is_last {
        format!("{}{}", indent, "    ")
    } else {
        format!("{}{}", indent, "│   ")
    };

    let total_items = directory.files.len() + directory.subdirectories.len();
    let mut current_item = 0;

    // Process files
    for file in &directory.files {
        current_item += 1;
        let is_last_item = current_item == total_items;
        let connector = if is_last_item { "└── " } else { "├── " };
        tree.push_str(&format!("{}{}{}\n", new_indent, connector, file));
    }

    // Process subdirectories
    for sub_dir in directory.subdirectories.values() {
        current_item += 1;
        let is_last_item = current_item == total_items;
        let sub_tree = directory_tree_to_string(sub_dir, &new_indent, is_last_item, false);
        tree.push_str(&sub_tree);
    }

    tree
}

/// Generates a tree-like structure of the project files.
///
/// # Arguments
///
/// * `project_name` - Name of the project.
/// * `files` - List of files with their paths.
/// * `project_root` - Path to the project root directory.
///
/// # Returns
///
/// * `Result<String>` - The tree structure as a string.
pub fn generate_tree(
    project_name: &str,
    files: &[(PathBuf, String)],
    project_root: &Path,
) -> Result<String> {
    let mut root = build_directory_tree(files, project_root);
    root.name = project_name.to_string();

    let tree = directory_tree_to_string(&root, "", true, true);
    Ok(tree)
}

```

