package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"unsafe"
)

func main() {
	// fmt.Printf("O tamanho de FATEntry e %d bytes \n", unsafe.Sizeof(FATEntry{})) -> 12 bytes, compilador adiciona 3 bytes apos o campo USED para alinha ao tamanho com os outros campos -> Facilita a busca e acesso em memoria
	fsSize := getFileSystemSize()
	if fsSize == 0 {
		return
	}
	var blockSize uint32 = 4096
	fs, err := createFileSystem(blockSize, fsSize)
	if err != nil {
		return
	}

	fs.operateFileSystem()
}

// Create File System

func getFileSystemSize() uint32 {
	var size uint32
	running := true
	for running {
		fmt.Println("Escolha sua opção:")
		fmt.Println("1. 10MB")
		fmt.Println("2. 100MB")
		fmt.Println("3. 800MB")
		fmt.Println("4. Sair.")
		consoleScanner := bufio.NewScanner(os.Stdin)
		fmt.Printf("Resposta: ")
		consoleScanner.Scan()
		inputStr := consoleScanner.Text()
		option, e := strconv.Atoi(inputStr)
		if e != nil {
			fmt.Printf("Entrada inválida: '%s'. Por favor, insira um número entre 1 e 4.\n", inputStr)
			continue
		}
		switch option {
		case 1:
			size = 10 * 1024 * 1024
		case 2:
			size = 100 * 1024 * 1024
		case 3:
			size = 800 * 1024 * 1024
		case 4:
			running = false
			continue
		default:
			fmt.Println("Opção inválida. Escolha um número entre 1 e 4.")
		}
		return size
	}
	return 0
}

type Header struct {
	TotalSize            uint32
	BlockSize            uint32
	FreeSpace            uint32
	FATEntrypointAddress uint32
	RootDirStart         uint32
	DataStart            uint32
}

type FATEntry struct {
	BlockID     uint32 // 4 bytes de 0 a 2**32 - 1
	NextBlockID uint32 // 4 bytes
	Used        bool   // 1 byte
}

type FileEntry struct {
	Name         [32]byte
	Path         [128]byte
	Size         uint32
	FirstBlockID uint32
	Protected    bool
	IsDirectory  bool
}
type FURGFileSystem struct {
	Header      Header
	FAT         []FATEntry
	RootDir     []FileEntry
	FilePointer *os.File
}

func calculateFATSize(FileSystemSize uint32, BlockSize uint32, FATEntrySize uint32) uint32 {
	totalBlocks := FileSystemSize / BlockSize
	fatSize := totalBlocks * FATEntrySize
	return fatSize
}

func calculateRootDirSize(entriesNumber uint32) uint32 {
	rootDirSize := uint32(entriesNumber) * uint32(unsafe.Sizeof(FileEntry{}))
	return rootDirSize
}

func calculateHeaderSize() uint32 {
	HeaderSize := uint32(unsafe.Sizeof(Header{}))
	return HeaderSize
}

func createFileSystem(BlockSize uint32, TotalSize uint32) (*FURGFileSystem, error) {
	f, err := os.OpenFile("furg.fs2", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("Erro ao abrir/criar o arquivo", err)
		return nil, nil
	} else {
		fmt.Println("Arquivo do FileSystem criado com sucesso com permissao de escrita e leitura.")
	}

	var entriesNumber uint32 = 100

	rootDirSize := calculateRootDirSize(entriesNumber)
	headerSize := calculateHeaderSize()
	fatEntrySize := uint32(unsafe.Sizeof(FATEntry{}))
	FATSize := calculateFATSize(TotalSize-headerSize-rootDirSize, BlockSize, fatEntrySize)

	header := Header{
		TotalSize:            TotalSize,
		BlockSize:            BlockSize,
		FreeSpace:            TotalSize - headerSize - FATSize - rootDirSize,
		FATEntrypointAddress: headerSize,
		RootDirStart:         headerSize + FATSize,
		DataStart:            headerSize + FATSize + rootDirSize,
	}

	err = binary.Write(f, binary.LittleEndian, header)
	if err != nil {
		fmt.Println("Escrita do arquivo em binario falhou.", err)
	}
	fileSystem := FURGFileSystem{
		Header:      header,
		FAT:         make([]FATEntry, FATSize/fatEntrySize),
		RootDir:     make([]FileEntry, entriesNumber),
		FilePointer: f,
	}

	return &fileSystem, nil // Retornar pontiero pois ao inves de duplicar a memoria, apenas retorna o ponteiro de referencia a ele.
}

// Operate in File System

func (fs *FURGFileSystem) operateFileSystem() {
	var option int
	for {
		fmt.Println("\n--- Menu do Sistema de Arquivos FURGfs2 ---")
		fmt.Println("1. Copiar arquivo para o sistema de arquivos")
		fmt.Println("2. Remover arquivo do sistema de arquivos")
		fmt.Println("3. Renomear arquivo armazenado no FURGfs2")
		fmt.Println("4. Listar todos os arquivos armazenados no FURGfs2")
		fmt.Println("5. Listar o espaço livre em relação ao total do FURGfs2")
		fmt.Println("6. Proteger/desproteger arquivo contra escrita/remoção")
		fmt.Println("7. Copiar um arquivo do sistema ficticio para o real")
		fmt.Println("8. Criar diretório")
		fmt.Println("0. Sair")
		fmt.Print("Escolha uma opção: ")
		fmt.Scanln(&option)

		switch option {
		case 1:
			var externalPath string
			var internalPath string
			var protectionBit int

			fmt.Println("Opção 1: Copiar arquivo para o sistema de arquivos.")

			fmt.Print("Digite o caminho completo do arquivo para copiar: ")
			fmt.Scanln(&externalPath)

			fmt.Print("Digite o caminho completo no FurgFS2 onde o arquivo vai ficar: (digite / para raiz) ")
			fmt.Scanln(&internalPath)

			fmt.Print("Digite o bit de proteção (1 para protegido, 0 para não protegido): ")
			fmt.Scanln(&protectionBit)

			if protectionBit != 0 && protectionBit != 1 {
				fmt.Println("Bit de proteção inválido! Deve ser 1 ou 0.")
				continue
			}
			isProtected := protectionBit == 1

			fs.CopyFileToFileSystem(externalPath, internalPath, isProtected)
		case 2:
			var fileName string
			var path string

			fmt.Println("Opção 2: Remover arquivo do sistema de arquivos.")

			fmt.Print("Digite o nome completo do arquivo(com extensão) para remover: ")
			fmt.Scanln(&fileName)

			fmt.Print("Digite o caminho do arquivo: ")
			fmt.Scanln(&path)

			fmt.Printf("Arquivo '%s' será removido.\n", fileName)
			err := fs.RemoveFileFromFileSystem(fileName, path)
			if err != nil {
				fmt.Println(err)
			}

		case 3:
			var oldName string
			var path string
			var newName string

			fmt.Println("Opção 3: Renomear arquivo armazenado no FURGfs2.")

			fmt.Print("Digite o o nome completo do arquivo(com extensão) a ser renomeado: ")
			fmt.Scanln(&oldName)

			fmt.Print("Digite o caminho do arquivo: ")
			fmt.Scanln(&path)

			fmt.Print("Digite o novo nome do arquivo: ")
			fmt.Scanln(&newName)

			fmt.Printf("Arquivo '%s' será renomeado para '%s'.\n", oldName, newName)
			err := fs.RenameFileFromFileSystem(oldName, path, newName)
			if err != nil {
				fmt.Println(err)
			}
		case 4:
			fmt.Println("Opção 4: Listar todos os arquivos armazenados no FURGfs2.")
			fmt.Println("Listagem de arquivos:")
			fs.ShowAllFilesFromFileSystem()
		case 5:
			fmt.Println("Opção 5: Listar o espaço livre em relação ao total do FURGfs2.")
			fmt.Println("Espaço livre e total:")
			fs.ShowFreeSpaceFromFileSystem()
		case 6:
			var fileName string
			var path string

			fmt.Println("Opção 6: Proteger/desproteger arquivo contra escrita/remoção.")

			fmt.Print("Digite o nome do arquivo a ser protegido/desprotegido: ")
			fmt.Scanln(&fileName)

			fmt.Print("Digite o caminho do arquivo: ")
			fmt.Scanln(&path)

			err := fs.ChangePermission(fileName, path)
			if err != nil {
				fmt.Println(err)
			}
		case 7:
			var fileName string
			var internalPath string
			var externalPath string

			fmt.Print("Digite o nome do arquivo que deseja copiar para o sistema real: ")
			fmt.Scanln(&fileName)

			if fileName == "" {
				fmt.Println("Erro: Nome do arquivo não pode estar vazio.")
				break
			}

			fmt.Print("Digite o caminho do arquivo no FURGfs2: ")
			fmt.Scanln(&internalPath)
			if internalPath == "" {
				fmt.Println("Erro: Caminho do arquivo não pode estar vazio.")
				break
			}

			fmt.Print("Digite o caminho completo onde deseja salvar o arquivo(lembra de colocar a extensao caso queira abrir o arquivo): ")
			fmt.Scanln(&externalPath)
			if externalPath == "" {
				fmt.Println("Erro: Caminho de destino não pode estar vazio.")
				break
			}

			err := fs.CopyFileFromFileSystem(fileName, internalPath, externalPath)
			if err != nil {
				fmt.Printf("Erro ao copiar o arquivo: %v\n", err)
			} else {
				fmt.Printf("Arquivo '%s' copiado com sucesso para '%s'.\n", fileName, externalPath)
			}
		case 8:
			fmt.Println("Opção 8: Criar diretório.")
			fmt.Print("Digite o nome do diretório a ser criado: ")
			var name string
			fmt.Scanln(&name)
			var path string
			fmt.Print("Digite o caminho do diretório pai: ")
			fmt.Scanln(&path)
			err := fs.CreateDirectory(name, path)

			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Printf("Diretório '%s' criado com sucesso no caminho '%s'.\n", name, path)
			}
		case 9:
			fmt.Println("Opção 9: Listar diretórios.")
			fs.Tree()
		case 0:
			fmt.Println("Saindo do sistema. Até logo!")
			return

		default:
			fmt.Println("Opção inválida. Tente novamente.")
		}
	}
}

func (fs *FURGFileSystem) CheckFileEntryAlreadyExists(name [32]byte, path [128]byte) int {
	fileNameStr := string(name[:])
	pathStr := string(path[:])

	for i, v := range fs.RootDir {
		existingFileName := string(v.Name[:])
		existingPath := string(v.Path[:])

		if existingFileName == fileNameStr && existingPath == pathStr {
			return i
		}
	}

	// Retorna -1 se o arquivo não for encontrado
	return -1
}

func (fs *FURGFileSystem) ProcessFileForFileSystem(path string) (*os.File, [32]byte, string, uint32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, [32]byte{}, "", 0, fmt.Errorf("erro ao abrir o arquivo: %w", err)
	}

	fileInfo, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, [32]byte{}, "", 0, fmt.Errorf("erro ao obter informações do arquivo: %w", err)
	}

	fileSize := fileInfo.Size()
	if fileSize > int64(fs.Header.FreeSpace) {
		f.Close()
		return nil, [32]byte{}, "", 0, fmt.Errorf("erro: o arquivo é muito grande para o espaço disponível")
	}

	var fileSizeUint32 uint32 = uint32(fileSize)

	fileName := filepath.Base(path)

	if len(fileName) > 32 {
		f.Close()
		return nil, [32]byte{}, "", 0, fmt.Errorf("erro: o nome do arquivo excede o limite de 32 bytes")
	}

	var fileNameArray [32]byte
	copy(fileNameArray[:], fileName)

	return f, fileNameArray, fileName, fileSizeUint32, nil
}

func (fs *FURGFileSystem) CopyFileToFileSystem(externalPath string, internalPath string, protected bool) bool {
	f, fileNameArray, fileName, fileSizeUint32, err := fs.ProcessFileForFileSystem(externalPath)

	if err != nil {
		fmt.Println(err)
		return false
	}

	var pathArray [128]byte
	copy(pathArray[:], internalPath)

	if cod := fs.CheckFileEntryAlreadyExists(fileNameArray, pathArray); cod != -1 {
		fmt.Println("erro: arquivo com o mesmo nome já existe no diretório pai.")
		return false
	}

	buf := make([]byte, fs.Header.BlockSize)

	var firstBlock, previousBlock uint32
	firstBlockSet := false
	for {
		bytesRead, err := f.Read(buf)
		if err != nil && err != io.EOF {
			fmt.Println("Erro ao ler o arquivo:", err)
			return false
		}
		if bytesRead == 0 {
			break
		}

		var currentBlockID uint32
		found := false
		for i, v := range fs.FAT {
			if !v.Used {
				currentBlockID = uint32(i)
				tmp := FATEntry{
					BlockID:     currentBlockID,
					NextBlockID: 0,
					Used:        true,
				}
				found = true
				fs.FAT[i] = tmp
				break
			}
		}
		if !found {
			fmt.Println("erro: espaço insuficiente na FAT.")
			return false
		}

		if !firstBlockSet {
			firstBlock = currentBlockID
			firstBlockSet = true
		} else {
			fs.FAT[previousBlock].NextBlockID = currentBlockID
		}
		previousBlock = currentBlockID

		_, err = fs.FilePointer.Seek(int64(fs.Header.DataStart+(currentBlockID*fs.Header.BlockSize)), 0)
		if err != nil {
			fmt.Println("Erro ao mover ponteiro do arquivo:", err)
			return false
		}
		_, err = fs.FilePointer.Write(buf[:bytesRead])
		if err != nil {
			fmt.Println("Erro ao escrever dados no arquivo:", err)
			return false
		}
	}

	for i, entry := range fs.RootDir {
		if entry.Name[0] == 0 {
			fs.RootDir[i] = FileEntry{
				Name:         fileNameArray,
				Path:         pathArray,
				Size:         fileSizeUint32,
				FirstBlockID: firstBlock,
				Protected:    protected,
			}
			break
		}
	}
	fs.Header.FreeSpace -= fileSizeUint32
	fmt.Printf("Arquivo '%s' copiado com sucesso para o sistema de arquivos.\n", fileName)
	defer f.Close()
	return true
}

func (fs *FURGFileSystem) CreateDirectory(name string, path string) error {
	var nameArray [32]byte
	copy(nameArray[:], name)

	if bytes.Contains(nameArray[:], []byte("/")) {
		return fmt.Errorf("erro: O nome do diretório não pode conter '/'")
	}

	if isAllNullBytes(name) {
		return fmt.Errorf("erro: Não existem diretórios com nome vazio")
	}

	// verificar se o path existe
	if !fs.CheckPathExists(path) {
		return fmt.Errorf("erro: O caminho '%s' não existe", path)
	}

	// cria entry file
	var pathArray [128]byte
	copy(pathArray[:], path)

	// verifica se já existe um diretório com o mesmo nome dentro do diretório pai
	if i := fs.CheckFileEntryAlreadyExists(nameArray, pathArray); i != -1 {
		return fmt.Errorf("erro: Já existe um diretório com o nome '%s' no diretório pai", name)
	}

	fileEntry := FileEntry{
		Name:        nameArray,
		Path:        pathArray,
		IsDirectory: true,
	}

	err := fs.AddFileEntry(fileEntry)
	if err != nil {
		return err
	}

	return nil
}

func (fs *FURGFileSystem) AddFileEntry(fileEntry FileEntry) error {
	for i, entry := range fs.RootDir {
		if entry.Name[0] == 0 {
			fs.RootDir[i] = fileEntry
			return nil
		}
	}
	return fmt.Errorf("erro: Não foi possível adicionar a entrada de arquivo ao sistema de arquivos")
}

func (fs *FURGFileSystem) CheckPathExists(path string) bool {
	if path == "/" {
		return true
	}

	for i, v := range fs.RootDir {
		trimmedExistingPath := string(bytes.Trim(v.Path[:], "\x00"))

		if path == trimmedExistingPath {
			fmt.Println("encontrou o path")
			if fs.RootDir[i].IsDirectory {
				return true
			}
		}
	}

	return false
}

func (fs *FURGFileSystem) Tree() {
	fmt.Println("/")
	// Começa listando os arquivos e diretórios sem pai (root)
	for i := range fs.RootDir {
		entry := &fs.RootDir[i]
		name := string(bytes.Trim(entry.Name[:], "\x00")) // Remove bytes nulos do nome
		path := string(bytes.Trim(entry.Path[:], "\x00")) // Remove bytes nulos do path
		if name != "" {
			if path == "/" {
				fmt.Printf("/%s (Size: %d bytes)\n", name, entry.Size)
			} else {
				fmt.Printf("%s/%s (Size: %d bytes)", path, name, entry.Size)
			}
		}

	}
}

func (fs *FURGFileSystem) RemoveFileFromFileSystem(fileName, path string) error {
	var fileNameArray [32]byte
	copy(fileNameArray[:], fileName)

	var pathArray [128]byte
	copy(pathArray[:], path)

	if isAllNullBytes(fileName) {
		return fmt.Errorf("erro: Não existem arquivos com nome vazio")
	}

	rootDirIndex := fs.CheckFileEntryAlreadyExists(fileNameArray, pathArray)
	if rootDirIndex == -1 {
		return fmt.Errorf("erro: O arquivo '%s' em '%s' não foi armazenado no sistema de arquivos", path, fileName)
	}

	f := fs.RootDir[rootDirIndex]

	if f.Protected {
		return fmt.Errorf("erro: Arquivo protegido, troque sua proteção para poder remover")
	}

	nextBlockId := f.FirstBlockID
	for nextBlockId != 0 {
		currentFileEntry := fs.FAT[nextBlockId]
		nextBlockId = currentFileEntry.NextBlockID
		fs.FAT[nextBlockId] = FATEntry{}
	}
	fs.Header.FreeSpace += fs.RootDir[rootDirIndex].Size

	fs.RootDir[rootDirIndex] = FileEntry{}

	fmt.Printf("O arquivo com nome '%s' em '%s' foi removido no sistema de arquivos.\n", fileName, path)
	return nil
}

func (fs *FURGFileSystem) RenameFileFromFileSystem(oldFileName, path, newFileName string) error {
	var oldFileNameArray [32]byte
	copy(oldFileNameArray[:], oldFileName)

	var pathArray [128]byte
	copy(pathArray[:], path)

	rootDirIndex := fs.CheckFileEntryAlreadyExists(oldFileNameArray, pathArray)
	if rootDirIndex == -1 {
		return fmt.Errorf("erro: O arquivo com nome '%s' não foi armazenado no sistema de arquivos", oldFileName)
	}

	var newFileNameArray [32]byte
	copy(newFileNameArray[:], newFileName)
	if fs.RootDir[rootDirIndex].Protected {
		return fmt.Errorf("erro: Arquivo protegido, troque sua proteção para poder remover")
	}
	fs.RootDir[rootDirIndex].Name = newFileNameArray

	fmt.Printf("arquivo '%s' renomeado, antes era '%s", newFileName, oldFileName)
	return nil
}

func isAllNullBytes(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != 0 {
			return false
		}
	}
	return true
}

func (fs *FURGFileSystem) ShowAllFilesFromFileSystem() {
	for i, file := range fs.RootDir {
		fileName := string(file.Name[:])
		path := string(file.Path[:])

		if fileName != "" && !isAllNullBytes(fileName) {
			fmt.Printf("%d. %s - path: %s", i, fileName, path)
			fmt.Printf("  -  %s\n", map[bool]string{true: "protegido", false: "desprotegido"}[file.Protected])
		}
	}
}

func (fs *FURGFileSystem) ShowFreeSpaceFromFileSystem() {
	totalSize := (fs.Header.TotalSize) / (1024 * 1024)
	freeSpace := (fs.Header.FreeSpace) / (1024 * 1024)

	occupiedSpace := totalSize - freeSpace
	percentOccupied := (float64(occupiedSpace) / float64(totalSize)) * 100

	fmt.Printf("Espaço total: %d MB\n", totalSize)
	fmt.Printf("Espaço livre: %d MB\n", freeSpace)
	fmt.Printf("Espaço ocupado: %d MB (%.2f%%)\n", occupiedSpace, percentOccupied)
}

func (fs *FURGFileSystem) ChangePermission(fileName, path string) error {
	var fileNameArray [32]byte
	copy(fileNameArray[:], fileName)

	var pathArray [128]byte
	copy(pathArray[:], path)

	if isAllNullBytes(fileName) {
		return fmt.Errorf("erro: Não existem arquivos com nome vazio")
	}

	rootDirIndex := fs.CheckFileEntryAlreadyExists(fileNameArray, pathArray)
	if rootDirIndex == -1 {
		return fmt.Errorf("erro: O arquivo com nome '%s' não foi armazenado no sistema de arquivos", fileName)
	}

	f := &fs.RootDir[rootDirIndex]
	fmt.Printf("Mudando a proteção do arquivo, agora é: '%s'\n", map[bool]string{true: "protegido", false: "desprotegido"}[f.Protected])
	f.Protected = !f.Protected

	if f.Protected {
		fmt.Printf("O arquivo '%s' agora está protegido.\n", fileName)
	} else {
		fmt.Printf("O arquivo '%s' agora está desprotegido.\n", fileName)
	}

	return nil
}

func (fs *FURGFileSystem) CopyFileFromFileSystem(fileName, internalPath, externalPath string) error {
	var fileNameArray [32]byte
	copy(fileNameArray[:], []byte(fileName))

	var internalPathArray [128]byte
	copy(internalPathArray[:], []byte(internalPath))

	// Verificar se o nome do arquivo é vazio
	if isAllNullBytes(fileName) {
		return fmt.Errorf("Erro: Não existem arquivos com nome vazio.")
	}

	// Localizar o arquivo no diretório raiz
	rootDirIndex := fs.CheckFileEntryAlreadyExists(fileNameArray, internalPathArray)
	if rootDirIndex == -1 {
		return fmt.Errorf("Erro: O arquivo com nome '%s' não foi encontrado no sistema de arquivos.\n", fileName)
	}

	fileEntry := fs.RootDir[rootDirIndex]

	destFile, err := os.Create(externalPath)
	if err != nil {
		return fmt.Errorf("Erro ao criar o arquivo no sistema real: %v", err)
	}
	defer destFile.Close()

	currentBlockID := fileEntry.FirstBlockID
	for currentBlockID != 0 {
		offset := int64(fs.Header.DataStart + (currentBlockID * fs.Header.BlockSize))
		_, err := fs.FilePointer.Seek(offset, 0)
		if err != nil {
			return fmt.Errorf("Erro ao mover ponteiro para bloco %d: %v", currentBlockID, err)
		}

		buf := make([]byte, fs.Header.BlockSize)
		bytesRead, err := fs.FilePointer.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("Erro ao ler bloco %d: %v", currentBlockID, err)
		}

		_, err = destFile.Write(buf[:bytesRead])
		if err != nil {
			return fmt.Errorf("Erro ao escrever dados no arquivo destino: %v", err)
		}

		currentBlockID = fs.FAT[currentBlockID].NextBlockID
	}

	fmt.Printf("Arquivo '%s' copiado com sucesso para o caminho '%s'.\n", fileName, externalPath)
	return nil
}
