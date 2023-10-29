"# vite rule"

load("@aspect_bazel_lib//lib:expand_make_vars.bzl", "expand_variables")
load("@aspect_bazel_lib//lib:copy_to_bin.bzl", "copy_file_to_bin_action", "copy_files_to_bin_actions")
load("@aspect_rules_js//js:libs.bzl", "js_lib_helpers")
load("@aspect_rules_js//js:providers.bzl", "JsInfo", "js_info")
load("@aspect_rules_ts//ts:defs.bzl", "TsConfigInfo")

_ATTRS = {
    "deps": attr.label_list(
        default = [],
        doc = "A list of direct dependencies that are required to build the bundle",
        providers = [JsInfo],
    ),
    "data": js_lib_helpers.JS_LIBRARY_DATA_ATTR,
    "entry_points": attr.label_list(
        allow_files = True,
        doc = """The bundle's entry points (e.g. your main.js or app.js or index.js)

Specify either this attribute or `entry_point`, but not both.
""",
    ),
    "srcs": attr.label_list(
        allow_files = True,
        default = [],
        doc = """Source files to be made available to esbuild""",
    ),
    "config": attr.label(
        mandatory = True,
        allow_single_file = True,
        doc = """Configuration file used for esbuild. Note that options set in this file may get overwritten. If you formerly used `args` from rules_nodejs' npm package `@bazel/esbuild`, replace it with this attribute.
        TODO: show how to write a config file that depends on plugins, similar to the esbuild_config macro in rules_nodejs.
    """,
    ),
    "tsconfig": attr.label(
        mandatory = True,
        allow_single_file = True,
        doc = """TypeScript configuration file used by esbuild. Default to an empty file with no configuration.
        
        See https://esbuild.github.io/api/#tsconfig for more details
    """,
    ),
}

_DEFAULT_VITE = "@npm//vite/bin:vite"

def _bin_relative_path(ctx, file):
    prefix = ctx.bin_dir.path + "/"
    if file.path.startswith(prefix):
        return file.path[len(prefix):]

    # Since file.path is relative to execroot, go up with ".." starting from
    # ctx.bin_dir until we reach execroot, then join that with the file path.
    up = "/".join([".." for _ in ctx.bin_dir.path.split("/")])
    return up + "/" + file.path


def _vite_project_impl(ctx):
    node_toolinfo = ctx.toolchains["@rules_nodejs//nodejs:toolchain_type"].nodeinfo

    entry_points = ctx.files.entry_points
    entry_points_bin_copy = copy_files_to_bin_actions(ctx, entry_points)
    entry_points_relative_paths = [_bin_relative_path(ctx, entry_point) for entry_point in entry_points_bin_copy]

    tsconfig_bin_copy = copy_file_to_bin_action(ctx, ctx.file.tsconfig)
    tsconfig_relative_path = _bin_relative_path(ctx, tsconfig_bin_copy)

    output_sources = []
    dist_dir = ctx.actions.declare_directory("dist")
    output_sources.extend([dist_dir])

    execution_requirements = {}
    if "no-remote-exec" in ctx.attr.tags:
        execution_requirements = {"no-remote-exec": "1"}

    args = ctx.actions.args()
    other_inputs = []

    config_bin_copy = copy_file_to_bin_action(ctx, ctx.file.config)
    other_inputs.append(config_bin_copy)
    args.add("--config=%s" % _bin_relative_path(ctx, config_bin_copy))
    args.add("build")

    input_sources = depset(
        copy_files_to_bin_actions(ctx, [
            file
            for file in ctx.files.srcs
            if not (file.path.endswith(".d.ts") or file.path.endswith(".tsbuildinfo"))
        ]) + entry_points_bin_copy + [tsconfig_bin_copy] + other_inputs + node_toolinfo.tool_files,
        transitive = [js_lib_helpers.gather_files_from_js_providers(
            targets = ctx.attr.srcs + ctx.attr.deps,
            include_transitive_sources = True,
            include_declarations = False,
            include_npm_linked_packages = True,
        )],
    )

    env = {
        "BAZEL_BINDIR": ctx.bin_dir.path,
    }

    launcher = _DEFAULT_VITE
    ctx.actions.run(
        inputs = input_sources,
        outputs = output_sources,
        arguments = args,
        progress_message = "Building Vite project %s" % (" ".join([_bin_relative_path(ctx, entry_point) for entry_point in entry_points])),
        execution_requirements = execution_requirements,
        mnemonic = "vite",
        env = env,
        executable = launcher,
    )

    npm_linked_packages = js_lib_helpers.gather_npm_linked_packages(
        srcs = ctx.attr.srcs,
        deps = [],
    )

    npm_package_store_deps = js_lib_helpers.gather_npm_package_store_deps(
        # Since we're bundling, only propagate `data` npm packages to the direct dependencies of
        # downstream linked `npm_package` targets instead of the common `data` and `deps` pattern.
        targets = ctx.attr.data,
    )

    output_sources_depset = depset(output_sources)

    runfiles = js_lib_helpers.gather_runfiles(
        ctx = ctx,
        sources = output_sources_depset,
        data = ctx.attr.data,
        # Since we're bundling, we don't propogate any transitive runfiles from dependencies
        deps = [],
    )

    return [
        DefaultInfo(
            files = output_sources_depset,
            runfiles = runfiles,
        ),
        js_info(
            npm_linked_package_files = npm_linked_packages.direct_files,
            npm_linked_packages = npm_linked_packages.direct,
            npm_package_store_deps = npm_package_store_deps,
            sources = output_sources_depset,
            # Since we're bundling, we don't propogate linked npm packages from dependencies since
            # they are bundled and the dependencies are dropped. If a subset of linked npm
            # dependencies are not bundled it is up the the user to re-specify these in `data` if
            # they are runtime dependencies to progagate to binary rules or `srcs` if they are to be
            # propagated to downstream build targets.
            transitive_npm_linked_package_files = npm_linked_packages.direct_files,
            transitive_npm_linked_packages = npm_linked_packages.direct,
            # Since we're bundling, we don't propogate any transitive sources from dependencies
            transitive_sources = output_sources_depset,
        ),
    ]

lib = struct(
    attrs = _ATTRS,
    implementation = _vite_project_impl,
    toolchains = [
        "@rules_nodejs//nodejs:toolchain_type",
    ],
)

vite_project = rule(
    implementation = _vite_project_impl,
    attrs = _ATTRS,
    toolchains = lib.toolchains,
    doc = """\
Runs Vite in Bazel
""",
)
