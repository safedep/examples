# Direct function calls without any
# variable assignment

def func1():
    print("func1 is called")

def func2():
    print("func2 is called")

def func3():
    func1()
    func2()

def main():
    func3()

if __name__ == "__main__":
    main()
