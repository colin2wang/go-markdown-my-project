package symbol

import "testing"

func TestGoExtract(t *testing.T) {
	src := `package main

import "fmt"

type Server struct {
	Port int
}

type Handler interface {
	Handle()
}

func main() {
	fmt.Println("hi")
}

func (s *Server) Start(ctx context.Context) error {
	return nil
}
`
	syms := Extract("go", src)
	kinds := map[string][]string{}
	for _, s := range syms {
		kinds[s.Kind] = append(kinds[s.Kind], s.Name)
	}
	assertContains(t, "func", kinds, "main")
	assertContains(t, "method", kinds, "Start")
	assertContains(t, "struct", kinds, "Server")
	assertContains(t, "interface", kinds, "Handler")

	if s, ok := findSym(syms, "method", "Start"); !ok || s.Parent != "Server" {
		t.Errorf("Start 应归属 Server，got %+v", s)
	}
}

func TestPythonExtract(t *testing.T) {
	src := `class Reader:
    def __init__(self):
        pass

    async def read_all(self, path):
        pass

def helper():
    pass
`
	syms := Extract("python", src)
	names := []string{}
	for _, s := range syms {
		names = append(names, s.Kind+":"+s.Name)
	}
	assertContains(t, "class", mapName(syms), "Reader")
	assertContains(t, "method", mapName(syms), "read_all") // 类内 def → method
	assertContains(t, "func", mapName(syms), "helper")
}

func TestJSExtract(t *testing.T) {
	src := `export function calculate(a, b) {
  return a + b;
}

export default class App {
  constructor() {}

  async render() {
  }
}

const helper = () => {};
`
	syms := Extract("typescript", src)
	m := mapName(syms)
	assertContains(t, "func", m, "calculate")
	assertContains(t, "class", m, "App")
	assertContains(t, "method", m, "render")
}

func TestRustExtract(t *testing.T) {
	src := `pub struct Config {
    name: String,
}

pub trait Processor {
    fn process(&self);
}

pub enum Mode {
    Full,
}

pub async fn run(cfg: Config) -> Result<()> {
    Ok(())
}

fn helper() {}
`
	syms := Extract("Rust", src)
	m := mapName(syms)
	assertContains(t, "struct", m, "Config")
	assertContains(t, "trait", m, "Processor")
	assertContains(t, "enum", m, "Mode")
	assertContains(t, "func", m, "run")
	assertContains(t, "func", m, "helper")
}

func TestJavaExtract(t *testing.T) {
	src := `public class Application {
    private final String name;

    public String getName() {
        return name;
    }

    public static void main(String[] args) {
    }
}

public interface Service {
    void start();
}
`
	syms := Extract("Java", src)
	m := mapName(syms)
	assertContains(t, "class", m, "Application")
	assertContains(t, "method", m, "getName")
	assertContains(t, "method", m, "main")
}

func TestCExtract(t *testing.T) {
	src := `struct Point {
    int x;
};

int main(void) {
    return 0;
}

static size_t count_items(const char *path) {
    return 0;
}
`
	syms := Extract("C", src)
	m := mapName(syms)
	assertContains(t, "class", m, "Point")
	assertContains(t, "func", m, "main")
	assertContains(t, "func", m, "count_items")
}

func TestUnknownLanguageReturnsEmpty(t *testing.T) {
	if syms := Extract("cobol", "IDENTIFICATION DIVISION."); len(syms) != 0 {
		t.Fatalf("未知语言应返回空，got %d", len(syms))
	}
}

func mapName(syms []Symbol) map[string][]string {
	m := map[string][]string{}
	for _, s := range syms {
		m[s.Kind] = append(m[s.Kind], s.Name)
	}
	return m
}

func findSym(syms []Symbol, kind, name string) (Symbol, bool) {
	for _, s := range syms {
		if s.Kind == kind && s.Name == name {
			return s, true
		}
	}
	return Symbol{}, false
}

func assertContains(t *testing.T, kind string, m map[string][]string, name string) {
	t.Helper()
	for _, n := range m[kind] {
		if n == name {
			return
		}
	}
	t.Errorf("%s 中未找到 %s；实际 %v", kind, name, m[kind])
}
