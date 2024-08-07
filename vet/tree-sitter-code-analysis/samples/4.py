x = 10
class A:
    def func(self):
        print(x)

    class B:
        x = 20
        def func(self):
            print(self.x)

if __name__ == "__main__":
    a = A()
    a.func()
    b = A.B()
    b.func()
