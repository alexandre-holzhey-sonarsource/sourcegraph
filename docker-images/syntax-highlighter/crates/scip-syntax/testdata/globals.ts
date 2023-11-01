class MyClass {
    public_field: number
    #private_field: number
    private also_private_field: number

    public_method() {}
    #private_method() {}
    private also_private_method() {}
}

export const SiteAdmin: StoryFn = args => {
    const props = useMemo(() => commonProps(args), [args])
    return (
        <OpenByDefaultWrapper>
            {({ menuButtonRef }) => (
        <WebStory>
            {webProps => <UserNavItem {...props} {...webProps} menuButtonRef={menuButtonRef} />}
    </WebStory>
    )}
    </OpenByDefaultWrapper>
)
}

interface MyInterface {
    bruh: number,
    sayBruh(): void,
}

enum MyEnum {
    zig,
    rust,
    go,
}

var global1 = 0;
var global2;

function func() {
    var c;
    function inAnotherFunc() {
        var b;
        function inAnother() {
            var a;
        }
    }
}

var myObject = {
  myProperty: "value",

  myMethod: function() {},
  myArrow: () => {},
};
