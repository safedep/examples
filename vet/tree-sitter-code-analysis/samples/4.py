x = 10
class A:
    def func(self):
        print(x)
        return ["nothing"]

    class B:
        x = 20
        def func(self):
            print(self.x)
            return 0

if __name__ == "__main__":
    a = A()
    a.func()
    b = A.B()
    b.func()

    b = A()
    b.func()
