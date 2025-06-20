# nibl: Not In Binary Language
**beta quality: don't consider this stable!**

**nibl** is a text-centric static site generator (SSG) designed for writers, storytellers, and creators who want to build websites directly from their manuscripts. It uniquely combines a powerful interactive fiction (IF) engine with "editor's sugar" to create a seamless workflow from first draft to finished product.

## The Core Idea

At its heart, `nibl` is built on a simple philosophy: your text files are the single source of truth. It's designed to get out of your way, letting you focus on writing. `nibl` bridges the gap between a writer's manuscript and a live website, offering powerful tools without forcing you to leave your text editor.

-   **Text-Centric:** Write your content in plain text and Markdown. `nibl` transforms it into a full static website.
-   **Editor's Sugar (EditML):** `nibl` integrates **[EditML](https://github.com/verkaro/editml-go/blob/main/docs/EditML-Spec-v2.5.md)**, a markup language that lets you embed versioning, alternative phrasing, and comments directly into your text. This allows you to maintain a single file with a complete history of your creative process, which `nibl` can process into a clean, final version for publication.
-   **Interactive Fiction Engine:** `nibl` includes a built-in IF engine powered by **`bigif`**. This allows you to write complex, choice-based narratives using a simple, intuitive syntax right inside your content files.

## Features

-   **Static Site Generation:** Converts a directory of Markdown files into a complete, portable website.
-   **Integrated IF Engine:** Compile `.biff` files to create branching narratives with state tracking.
-   **EditML Processing:** Automatically processes EditML syntax to generate clean, readable output from your drafts.
-   **Live-Reload Dev Server:** A built-in server watches for changes and automatically rebuilds your site, giving you an instant preview.
-   **Flexible Content Structure:** Generate content from a master story file or write individual pages.
-   **Simple Scaffolding:** Quickly create a new site or a new piece of content with `new` commands.

## Getting Started

1.  **Create a new site:**
    ```bash
    nibl new site my-new-story
    cd my-new-story
    ```

2.  **Write your story:**
    Edit the `site.biff` file to create your interactive narrative.

    ```text
    // title: My First Story
    // author: A. Writer

    === start ===
    // title: The Beginning
    This is the first page. {+This is a new addition!+}
    * Go to the next room -> room_2

    === room_2 ===
    // title: The Second Room
    You've reached the second room.
    {-This text was found lacking.-}{+This is the new version.+}
    * Go back -> start
    ```

3.  **Compile the story and build the site:**
    The `story` command reads your `.biff` file, processes the EditML, and generates the final HTML pages.
    ```bash
    nibl story
    ```

4.  **Serve it locally:**
    Start the development server to see your site in action.
    ```bash
    nibl serve
    ```
    Now, open your browser to `http://localhost:1313`.

## Why "Not In Binary Language"?

The name reflects the project's commitment to human-readable, plain-text formats. It's a generator for people who think in words, not in code.

---

This project is currently under active development. Contributions and feedback are welcome!

